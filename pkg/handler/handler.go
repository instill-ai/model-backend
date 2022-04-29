package handler

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/inferenceserver"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"

	database "github.com/instill-ai/model-backend/internal/db"
	metadataUtil "github.com/instill-ai/model-backend/internal/grpc/metadata"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

type FileMeta struct {
	path  string
	fInfo os.FileInfo
}

type handler struct {
	modelPB.UnimplementedModelServiceServer
	service service.Service
	triton  triton.Triton
}

func NewHandler(s service.Service, t triton.Triton) modelPB.ModelServiceServer {
	return &handler{
		service: s,
		triton:  t,
	}
}

//writeToFp takes in a file pointer and byte array and writes the byte array into the file
//returns error if pointer is nil or error in writing to file
func writeToFp(fp *os.File, data []byte) error {
	w := 0
	n := len(data)
	for {

		nw, err := fp.Write(data[w:])
		if err != nil {
			return err
		}
		w += nw
		if nw >= n {
			return nil
		}
	}
}

func updateConfigModelName(filePath string, oldModelName string, newModelName string) error {
	regStr := fmt.Sprintf("name:\\s+\"%v\"", oldModelName)
	nameRegx := regexp.MustCompile(regStr)
	fileData, _ := ioutil.ReadFile(filePath)
	fileString := string(fileData)
	fileString = nameRegx.ReplaceAllString(fileString, fmt.Sprintf("name: \"%v\"", newModelName))
	fileData = []byte(fileString)
	return ioutil.WriteFile(filePath, fileData, 0o600)
}

func isEnsembleConfig(configPath string) bool {
	fileData, _ := ioutil.ReadFile(configPath)
	fileString := string(fileData)
	return strings.Contains(fileString, "platform: \"ensemble\"")
}

func unzip(filePath string, dstDir string, namespace string, uploadedModel *datamodel.Model) (string, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return "", err
	}
	defer archive.Close()
	var readmeFilePath string
	var createdTModels []datamodel.TritonModel
	var currentNewModelName string
	var currentOldModelName string
	var ensembleFilePath string
	var newModelNameMap = make(map[string]string)
	for _, f := range archive.File {
		if strings.Contains(f.Name, "__MACOSX") || strings.Contains(f.Name, "__pycache__") { // ignore temp directory in macos
			continue
		}
		filePath := filepath.Join(dstDir, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return "", fmt.Errorf("invalid file path")
		}
		if f.FileInfo().IsDir() {
			dirName := f.Name
			if string(dirName[len(dirName)-1]) == "/" {
				dirName = dirName[:len(dirName)-1]
			}
			if !strings.Contains(dirName, "/") { // top directory model
				currentOldModelName = dirName
				dirName = fmt.Sprintf("%v#%v#%v#%v", namespace, uploadedModel.Name, dirName, uploadedModel.Instances[0].Name)
				currentNewModelName = dirName
				newModelNameMap[currentOldModelName] = currentNewModelName
			} else { // version folder
				dirName = strings.Replace(dirName, currentOldModelName, currentNewModelName, 1)
				patternVersionFolder := fmt.Sprintf("^%v/[0-9]+$", currentNewModelName)
				match, _ := regexp.MatchString(patternVersionFolder, dirName)
				if match {
					elems := strings.Split(dirName, "/")
					sVersion := elems[len(elems)-1]
					iVersion, err := strconv.ParseInt(sVersion, 10, 32)
					if err == nil {
						createdTModels = append(createdTModels, datamodel.TritonModel{
							Name:    currentNewModelName, // Triton model name
							Status:  modelPB.ModelInstance_STATUS_OFFLINE.String(),
							Version: int(iVersion),
						})
					}
				}
			}
			filePath := filepath.Join(dstDir, dirName)
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		// Update triton folder into format {model_name}#{task_name}#{task_version}
		subStrs := strings.Split(f.Name, "/")
		if len(subStrs) < 1 {
			continue
		}
		// Triton modelname is folder name
		oldModelName := subStrs[0]
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", namespace, uploadedModel.Name, subStrs[0], uploadedModel.Instances[0].Name)
		newModelName := subStrs[0]
		filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))
		if strings.Contains(f.Name, "README.md") {
			readmeFilePath = filePath
		}
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}
		fileInArchive, err := f.Open()
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return "", err
		}

		dstFile.Close()
		fileInArchive.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(filePath)
		if fileExtension == ".pbtxt" {
			if isEnsembleConfig(filePath) {
				ensembleFilePath = filePath
			}
			err = updateConfigModelName(filePath, oldModelName, newModelName)
			if err != nil {
				return "", err
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = updateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return "", err
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	uploadedModel.TritonModels = createdTModels
	return readmeFilePath, nil
}

// modelDir and dstDir are absolute path
func updateModelPath(modelDir string, dstDir string, namespace string, uploadedModel *datamodel.Model) (string, error) {
	var createdTModels []datamodel.TritonModel
	var ensembleFilePath string
	var newModelNameMap = make(map[string]string)
	var readmeFilePath string
	files := []FileMeta{}
	err := filepath.Walk(modelDir, func(path string, f os.FileInfo, err error) error {
		if !strings.Contains(path, ".git") {
			files = append(files, FileMeta{
				path:  path,
				fInfo: f,
			})
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		// Update triton folder into format {model_name}#{task_name}#{task_version}
		subStrs := strings.Split(strings.Replace(f.path, modelDir+"/", "", 1), "/")
		if len(subStrs) < 1 {
			continue
		}
		// Triton modelname is folder name
		oldModelName := subStrs[0]
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", namespace, uploadedModel.Name, oldModelName, uploadedModel.Instances[0].Name)
		var filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))
		if f.fInfo.IsDir() { // create new folder
			_ = os.Mkdir(filePath, os.ModePerm)
			newModelNameMap[oldModelName] = subStrs[0]
			if v, err := strconv.Atoi(subStrs[len(subStrs)-1]); err == nil {
				createdTModels = append(createdTModels, datamodel.TritonModel{
					Name:    subStrs[0], // Triton model name
					Status:  modelPB.ModelInstance_STATUS_OFFLINE.String(),
					Version: int(v),
				})
			}
			continue
		}
		if strings.Contains(filePath, "README") {
			readmeFilePath = filePath
		}
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.fInfo.Mode())
		if err != nil {
			log.Fatal(err)
		}
		srcFile, err := os.Open(f.path)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			log.Fatal(err)
		}
		dstFile.Close()
		srcFile.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(filePath)
		if fileExtension == ".pbtxt" {
			if isEnsembleConfig(filePath) {
				ensembleFilePath = filePath
			}
			err = updateConfigModelName(filePath, oldModelName, subStrs[0])
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = updateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				log.Fatal(err)
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	uploadedModel.TritonModels = createdTModels
	return readmeFilePath, nil
}

func saveFile(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (outFile string, modelInfo *datamodel.Model, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateModelBinaryFileUploadRequest

	var tmpFile string

	var uploadedModel datamodel.Model
	for {
		fileData, err = stream.Recv() //ignoring the data  TO-Do save files received
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed unexpectedly while reading chunks from stream")
			return "", &datamodel.Model{}, err
		}

		if firstChunk { //first chunk contains file name
			tmpFile = path.Join("/tmp", uuid.New().String()+".zip")
			fp, err = os.Create(tmpFile)
			visibility := modelPB.ModelDefinition_VISIBILITY_PRIVATE.String()
			if fileData.Visibility == modelPB.ModelDefinition_VISIBILITY_PUBLIC {
				visibility = modelPB.ModelDefinition_VISIBILITY_PUBLIC.String()
			}
			uploadedModel = datamodel.Model{
				Name:        fileData.Name,
				Visibility:  visibility,
				Source:      modelPB.ModelDefinition_SOURCE_LOCAL.String(),
				Description: fileData.Description,
				Instances: []datamodel.Instance{{
					Status: datamodel.ValidStatus(modelPB.ModelInstance_STATUS_OFFLINE.String()),
					Name:   "latest",
				}},
			}
			if err != nil {
				return "", &datamodel.Model{}, err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = writeToFp(fp, fileData.Bytes)
		if err != nil {
			return "", &datamodel.Model{}, err
		}
	}
	return tmpFile, &uploadedModel, nil
}

func savePredictInputs(stream modelPB.ModelService_TriggerModelBinaryFileUploadServer) (imageBytes [][]byte, modelId string, instanceName string, err error) {
	var firstChunk = true
	var fileData *modelPB.TriggerModelBinaryFileUploadRequest

	var allContentFiles []byte
	var length_of_files []uint64
	for {
		fileData, err = stream.Recv() //ignoring the data  TO-Do save files received
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return [][]byte{}, "", "", err
		}

		if firstChunk { //first chunk contains file name
			modelId = fileData.ModelName
			instanceName = fileData.InstanceName
			length_of_files = fileData.FileLengths

			firstChunk = false
		}
		allContentFiles = append(allContentFiles, fileData.Bytes...)
	}

	if len(length_of_files) == 0 {
		return [][]byte{}, "", "", fmt.Errorf("Wrong parameter length of files")
	}
	start := uint64(0)
	for i := 0; i < len(length_of_files); i++ {
		imageBytes = append(imageBytes, allContentFiles[start:start+length_of_files[i]])
		start = length_of_files[i]
	}
	return imageBytes, modelId, instanceName, nil
}

func makeError(statusCode codes.Code, title string, detail string) error {
	err := &datamodel.Error{
		Title:  title,
		Detail: detail,
	}
	data, _ := json.Marshal(err)
	return status.Error(statusCode, string(data))
}

func makeJsonResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

func getUsername(ctx context.Context) (string, error) {
	if metadatas, ok := metadataUtil.ExtractFromMetadata(ctx, "Username"); ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "Username not found in your request")
		}
		return metadatas[0], nil
	} else {
		return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
	}
}

func (s *handler) Liveness(ctx context.Context, pb *modelPB.LivenessRequest) (*modelPB.LivenessResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.LivenessResponse{Status: modelPB.LivenessResponse_SERVING_STATUS_NOT_SERVING}, nil
	}

	return &modelPB.LivenessResponse{Status: modelPB.LivenessResponse_SERVING_STATUS_SERVING}, nil
}

func (s *handler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.ReadinessResponse{Status: modelPB.ReadinessResponse_SERVING_STATUS_NOT_SERVING}, nil
	}

	return &modelPB.ReadinessResponse{Status: modelPB.ReadinessResponse_SERVING_STATUS_SERVING}, nil
}

func HandleCreateModelByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		if username == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}

		if strings.Contains(username, "..") || strings.Contains(username, "/") { //TODO add github username validator
			makeJsonResponse(w, 422, "Username error", "The user name should not contain special characters")
			return
		}

		modelName := r.FormValue("name")
		if modelName == "" {
			makeJsonResponse(w, 400, "Missing parameter", "Model name need to be specified")
			return
		}
		if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", modelName); !match {
			makeJsonResponse(w, 400, "Invalid parameter", "Model name is invalid")
			return
		}

		visibility := r.FormValue("visibility")
		if visibility != "" {
			if util.Visibility[visibility] == "" {
				makeJsonResponse(w, 400, "Invalid parameter", "Visibility is invalid")
				return
			} else {
				visibility = util.Visibility[visibility]
			}
		} else {
			visibility = modelPB.ModelDefinition_VISIBILITY_PRIVATE.String()
		}

		err := r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJsonResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}
		file, _, err := r.FormFile("content")
		if err != nil {
			makeJsonResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		buf := bytes.NewBuffer(make([]byte, 0))
		part := make([]byte, 1024)
		count := 0
		for {
			if count, err = reader.Read(part); err != nil {
				break
			}
			buf.Write(part[:count])
		}
		if err != io.EOF {
			makeJsonResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		tmpFile := path.Join("/tmp", uuid.New().String())
		fp, err := os.Create(tmpFile)
		if err != nil {
			makeJsonResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		err = writeToFp(fp, buf.Bytes())
		if err != nil {
			makeJsonResponse(w, 400, "File Error", "Error reading input file")
			return
		}

		owner, _ := json.Marshal(datamodel.Owner{
			Username: username,
			Type:     util.TYPE_USER,
			ID:       util.USER_ID,
		})
		var uploadedModel = datamodel.Model{
			Instances: []datamodel.Instance{{
				Status: datamodel.ValidStatus(modelPB.ModelInstance_STATUS_OFFLINE.String()),
				Name:   "latest",
			}},
			Name:        modelName,
			Namespace:   username,
			Visibility:  visibility,
			Source:      modelPB.ModelDefinition_SOURCE_LOCAL.String(),
			Owner:       owner,
			Description: r.FormValue("description"),
		}

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		modelService := service.NewService(modelRepository, tritonService)

		_, err = modelService.GetModelByName(username, uploadedModel.Name)
		if err == nil {
			makeJsonResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.Name))
			return
		}

		readmeFilePath, err := unzip(tmpFile, configs.Config.TritonServer.ModelStore, username, &uploadedModel)
		_ = os.Remove(tmpFile) // remove uploaded temporary zip file
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
			makeJsonResponse(w, 400, "Add Model Error", err.Error())
			return
		}
		if _, err := os.Stat(readmeFilePath); err == nil {
			modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
			if err != nil {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
				makeJsonResponse(w, 400, "Add Model Error", err.Error())
				return
			}
			if modelMeta.Task == "" {
				uploadedModel.Instances[0].Task = 0
			} else {
				if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
					uploadedModel.Instances[0].Task = uint(val)
				} else {
					util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
					makeJsonResponse(w, 400, "Add Model Error", "README.md contains unsupported task")
					return
				}
			}
		} else {
			uploadedModel.Instances[0].Task = 0
		}

		resModel, err := modelService.CreateModelBinaryFileUpload(username, &uploadedModel)
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
			makeJsonResponse(w, 500, "Add Model Error", err.Error())
			return
		}
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		m := jsonpb.Marshaler{OrigName: true, EnumsAsInts: false, EmitDefaults: true}
		var buffer bytes.Buffer
		err = m.Marshal(&buffer, &modelPB.CreateModelBinaryFileUploadResponse{Model: resModel})
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
			makeJsonResponse(w, 500, "Add Model Error", err.Error())
			return
		}
		_, _ = w.Write(buffer.Bytes())
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

// AddModel - upload a model to the model server
func (s *handler) CreateModelBinaryFileUpload(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (err error) {
	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}
	tmpFile, uploadedModel, err := saveFile(stream)
	if err != nil {
		return makeError(codes.InvalidArgument, "Save File Error", err.Error())
	}
	_, err = s.service.GetModelByName(username, uploadedModel.Name)
	if err == nil {
		return makeError(codes.AlreadyExists, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.Name))
	}

	uploadedModel.Namespace = username
	owner, _ := json.Marshal(datamodel.Owner{
		Username: username,
		Type:     util.TYPE_USER,
		ID:       util.USER_ID,
	})
	uploadedModel.Owner = owner

	// extract zip file from tmp to models directory
	readmeFilePath, err := unzip(tmpFile, configs.Config.TritonServer.ModelStore, username, uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
		return makeError(codes.InvalidArgument, "Save File Error", err.Error())
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
			return makeError(codes.InvalidArgument, "Add Model Error", err.Error())
		}
		if modelMeta.Task == "" {
			uploadedModel.Instances[0].Task = 0
		} else {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Instances[0].Task = uint(val)
			} else {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
				return makeError(codes.InvalidArgument, "Add Model Error", "README.md contains unsupported task")
			}
		}
	} else {
		uploadedModel.Instances[0].Task = 0
	}

	resModel, err := s.service.CreateModelBinaryFileUpload(username, uploadedModel)
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, uploadedModel.Name, uploadedModel.Instances[0].Name)
		return err
	}
	err = stream.SendAndClose(&modelPB.CreateModelBinaryFileUploadResponse{Model: resModel})
	if err != nil {
		return makeError(codes.Internal, "Add Model Error", err.Error())
	}

	return
}

func (s *handler) CreateModelByGitHub(ctx context.Context, in *modelPB.CreateModelByGitHubRequest) (*modelPB.CreateModelByGitHubResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.CreateModelByGitHubResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	// Validate the naming rule of model
	if match, _ := regexp.MatchString(util.MODEL_NAME_REGEX, in.Name); !match {
		return &modelPB.CreateModelByGitHubResponse{}, status.Error(codes.FailedPrecondition, "The name of model is invalid")
	}
	if in.Github == nil || in.Github.Repo == "" || !util.IsGitHubURL(in.Github.Repo) {
		return &modelPB.CreateModelByGitHubResponse{}, makeError(codes.FailedPrecondition, "Add Model Error", "Invalid GitHub URL")
	}

	modelSrcDir := fmt.Sprintf("/tmp/%v", uuid.New().String())
	github := datamodel.InstanceConfiguration{
		Repo: in.Github.Repo,
		Tag:  in.Github.Tag,
	}
	err = util.GitHubClone(modelSrcDir, github)
	if err != nil {
		return &modelPB.CreateModelByGitHubResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	githubInfo, err := util.GetGitHubRepoInfo(in.Github.Repo)
	if err != nil {
		return &modelPB.CreateModelByGitHubResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	visibility := util.Visibility[githubInfo.Visibility]
	if in.Visibility == modelPB.ModelDefinition_VISIBILITY_PUBLIC {
		visibility = modelPB.ModelDefinition_VISIBILITY_PUBLIC.String()
	} else if in.Visibility == modelPB.ModelDefinition_VISIBILITY_PRIVATE {
		visibility = modelPB.ModelDefinition_VISIBILITY_PRIVATE.String()
	}

	owner, _ := json.Marshal(datamodel.Owner{
		Username: username,
		Type:     util.TYPE_USER,
		ID:       util.USER_ID,
	})
	githubConfigObj, _ := json.Marshal(github)
	githubModelConfig, _ := json.Marshal(datamodel.ModelConfiguration{
		Repo: in.Github.Repo,
	})
	githubModel := datamodel.Model{
		Name:       in.Name,
		Namespace:  username,
		Source:     modelPB.ModelDefinition_SOURCE_GITHUB.String(),
		Visibility: visibility,
		Owner:      owner,
		Config:     githubModelConfig,
		Instances: []datamodel.Instance{{
			Name:   in.Github.Tag,
			Status: datamodel.ValidStatus(modelPB.ModelInstance_STATUS_OFFLINE.String()),
			Config: githubConfigObj,
		}},
	}

	_, err = s.service.GetModelByName(username, in.Name)
	if err == nil {
		return &modelPB.CreateModelByGitHubResponse{}, fmt.Errorf("The model %v already existed", githubModel.Name)
	}

	readmeFilePath, err := updateModelPath(modelSrcDir, configs.Config.TritonServer.ModelStore, username, &githubModel)
	_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, githubModel.Name, githubModel.Instances[0].Name)
		return &modelPB.CreateModelByGitHubResponse{}, err
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil || modelMeta.Task == "" {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, githubModel.Name, githubModel.Instances[0].Name)
			return &modelPB.CreateModelByGitHubResponse{}, err
		}
		if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			githubModel.Instances[0].Task = uint(val)
		} else {
			if modelMeta.Task != "" {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, githubModel.Name, githubModel.Instances[0].Name)
				return &modelPB.CreateModelByGitHubResponse{}, makeError(codes.InvalidArgument, "Add Model Error", "README.md contains unsupported task")
			} else {
				githubModel.Instances[0].Task = 0
			}
		}
	} else {
		githubModel.Instances[0].Task = 0
	}
	resModel, err := s.service.CreateModelBinaryFileUpload(username, &githubModel)

	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, username, githubModel.Name, githubModel.Instances[0].Name)
		return &modelPB.CreateModelByGitHubResponse{}, err
	}

	return &modelPB.CreateModelByGitHubResponse{Model: resModel}, nil
}

func (s *handler) UpdateModelInstance(ctx context.Context, in *modelPB.UpdateModelInstanceRequest) (*modelPB.UpdateModelInstanceResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.UpdateModelInstanceResponse{}, makeError(503, "LoadModel Error", "Triton Server not ready yet")
	}

	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.UpdateModelInstanceResponse{}, err
	}
	modelInstance, err := s.service.UpdateModelInstance(username, in)
	return &modelPB.UpdateModelInstanceResponse{Instance: modelInstance}, err
}

func (s *handler) ListModel(ctx context.Context, in *modelPB.ListModelRequest) (*modelPB.ListModelResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.ListModelResponse{}, err
	}

	resModels, err := s.service.ListModels(username)
	return &modelPB.ListModelResponse{Models: resModels}, err
}

func (s *handler) TriggerModel(ctx context.Context, in *modelPB.TriggerModelRequest) (*modelPB.TriggerModelResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, err
	}

	modelInDB, err := s.service.GetModelByName(username, in.ModelName)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.NotFound, "PredictModel", fmt.Sprintf("The model named %v not found in server", in.ModelName))
	}

	modelInstanceInDB, err := s.service.GetModelInstance(modelInDB.ID, in.InstanceName)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.NotFound, "PredictModel", fmt.Sprintf("The model %v  with instance %v not found in server", in.ModelName, in.InstanceName))
	}

	imgsBytes, _, err := ParseImageRequestInputsToBytes(in)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.InvalidArgument, "PredictModel", err.Error())
	}
	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := s.service.ModelInfer(username, in.ModelName, in.InstanceName, imgsBytes, task)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.InvalidArgument, "PredictModel", err.Error())
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return &modelPB.TriggerModelResponse{}, makeError(codes.Internal, "PredictModel", err.Error())
		}
	case modelPB.ModelInstance_TASK_DETECTION:
		b, err = json.Marshal(response.(*modelPB.DetectionOutputs))
		if err != nil {
			return &modelPB.TriggerModelResponse{}, makeError(codes.Internal, "PredictModel", err.Error())
		}
	default:
		b, err = json.Marshal(response.(*inferenceserver.ModelInferResponse))
		if err != nil {
			return &modelPB.TriggerModelResponse{}, makeError(codes.Internal, "PredictModel", err.Error())
		}
	}
	err = protojson.Unmarshal(b, data)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.Internal, "PredictModel", err.Error())
	}

	return &modelPB.TriggerModelResponse{Output: data}, nil
}

func (s *handler) TriggerModelBinaryFileUpload(stream modelPB.ModelService_TriggerModelBinaryFileUploadServer) error {
	if !s.triton.IsTritonServerReady() {
		return makeError(503, "PredictModel", "Triton Server not ready yet")
	}

	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}

	imageBytes, modelName, instanceName, err := savePredictInputs(stream)
	if err != nil {
		return makeError(500, "PredictModel", "Could not save the file")
	}

	modelInDB, err := s.service.GetModelByName(username, modelName)
	if err != nil {
		return makeError(404, "PredictModel", fmt.Sprintf("The model %v do not exist", modelName))
	}
	modelInstanceInDB, err := s.service.GetModelInstance(modelInDB.ID, instanceName)
	if err != nil {
		return makeError(404, "PredictModel", fmt.Sprintf("The model instance %v do not exist", instanceName))
	}
	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := s.service.ModelInfer(username, modelName, instanceName, imageBytes, task)
	if err != nil {
		return err
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}
	case modelPB.ModelInstance_TASK_DETECTION:
		b, err = json.Marshal(response.(*modelPB.DetectionOutputs))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}
	default:
		b, err = json.Marshal(response.(*inferenceserver.ModelInferResponse))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}
	}
	err = protojson.Unmarshal(b, data)
	if err != nil {
		return makeError(500, "PredictModel", err.Error())
	}
	err = stream.SendAndClose(&modelPB.TriggerModelBinaryFileUploadResponse{Output: data})
	return err
}

func HandlePredictModelByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		modelName := pathParams["model_name"]

		if username == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}
		if modelName == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter mode name not found")
			return
		}

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()

		modelService := service.NewService(modelRepository, tritonService)

		modelInDB, err := modelService.GetModelByName(username, modelName)
		if err != nil {
			makeJsonResponse(w, 404, "Model not found", "The model not found in server")
			return
		}

		modelInstanceName := pathParams["instance_name"]
		modelInstanceInDB, err := modelService.GetModelInstance(modelInDB.ID, modelInstanceName)
		if err != nil {
			makeJsonResponse(w, 404, "Model instance not found", "The model instance not found in server")
			return
		}

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJsonResponse(w, 400, "Internal Error", fmt.Sprintf("Error while reading file from request %v", err))
			return
		}

		imgsBytes, _, err := parseImageFormDataInputsToBytes(r)
		if err != nil {
			makeJsonResponse(w, 400, "File Input Error", err.Error())
			return
		}

		task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
		response, err := modelService.ModelInfer(username, modelName, modelInstanceName, imgsBytes, task)
		if err != nil {
			makeJsonResponse(w, 500, "Error Predict Model", err.Error())
			return
		}
		var data = &structpb.Struct{}
		var b []byte
		switch task {
		case modelPB.ModelInstance_TASK_CLASSIFICATION:
			b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
			if err != nil {
				makeJsonResponse(w, 500, "Error Predict Model", err.Error())
				return
			}
		case modelPB.ModelInstance_TASK_DETECTION:
			b, err = json.Marshal(response.(*modelPB.DetectionOutputs))
			if err != nil {
				makeJsonResponse(w, 500, "Error Predict Model", err.Error())
				return
			}
		default:
			b, err = json.Marshal(response.(*inferenceserver.ModelInferResponse))
			if err != nil {
				makeJsonResponse(w, 500, "Error Predict Model", err.Error())
				return
			}
		}
		err = protojson.Unmarshal(b, data)
		if err != nil {
			makeJsonResponse(w, 500, "Error Predict Model", err.Error())
			return
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		res, err := json.Marshal(&modelPB.TriggerModelBinaryFileUploadResponse{Output: data})
		if err != nil {
			makeJsonResponse(w, 500, "Error Predict Model", err.Error())
			return
		}
		_, _ = w.Write(res)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

func (s *handler) GetModel(ctx context.Context, in *modelPB.GetModelRequest) (*modelPB.GetModelResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	md, err := s.service.GetFullModelData(username, in.Name)
	return &modelPB.GetModelResponse{Model: md}, err
}

func (s *handler) DeleteModel(ctx context.Context, in *modelPB.DeleteModelRequest) (*modelPB.DeleteModelResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.DeleteModelResponse{}, err
	}
	return &modelPB.DeleteModelResponse{}, s.service.DeleteModel(username, in.Name)
}

func (s *handler) DeleteModelInstance(ctx context.Context, in *modelPB.DeleteModelInstanceRequest) (*modelPB.DeleteModelInstanceResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.DeleteModelInstanceResponse{}, err
	}
	return &modelPB.DeleteModelInstanceResponse{}, s.service.DeleteModelInstance(username, in.ModelName, in.InstanceName)
}
