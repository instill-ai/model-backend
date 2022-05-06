package handler

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
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

	"github.com/gofrs/uuid"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

func unzip(filePath string, dstDir string, owner string, uploadedModel *datamodel.Model) (string, error) {
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
				dirName = fmt.Sprintf("%v#%v#%v#%v", owner, uploadedModel.ID, dirName, uploadedModel.Instances[0].ID)
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
							State:   datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
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
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", owner, uploadedModel.ID, subStrs[0], uploadedModel.Instances[0].ID)
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
func updateModelPath(modelDir string, dstDir string, owner string, model *datamodel.Model) (string, error) {
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
		if f.path == modelDir {
			continue
		}
		// Update triton folder into format {model_name}#{task_name}#{task_version}
		subStrs := strings.Split(strings.Replace(f.path, modelDir+"/", "", 1), "/")
		if len(subStrs) < 1 {
			continue
		}
		// Triton modelname is folder name
		oldModelName := subStrs[0]
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", owner, model.ID, oldModelName, model.Instances[0].ID)
		var filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))

		if f.fInfo.IsDir() { // create new folder
			err = os.Mkdir(filePath, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			newModelNameMap[oldModelName] = subStrs[0]
			if v, err := strconv.Atoi(subStrs[len(subStrs)-1]); err == nil {
				createdTModels = append(createdTModels, datamodel.TritonModel{
					Name:    subStrs[0], // Triton model name
					State:   datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
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
	model.TritonModels = createdTModels
	return readmeFilePath, nil
}

func saveFile(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (outFile string, modelInfo *datamodel.Model, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateModelBinaryFileUploadRequest

	var tmpFile string

	var uploadedModel datamodel.Model
	for {
		fileData, err = stream.Recv()
		if fileData.Model == nil {
			return "", &datamodel.Model{}, fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", &datamodel.Model{}, fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { //first chunk contains file name
			rdid, _ := uuid.NewV4()
			tmpFile = path.Join("/tmp", rdid.String()+".zip")
			fp, err = os.Create(tmpFile)
			visibility := modelPB.Model_VISIBILITY_PRIVATE
			if fileData.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
				visibility = modelPB.Model_VISIBILITY_PUBLIC
			}
			var description = ""
			if fileData.Model.Description != nil {
				description = *fileData.Model.Description
			}
			modelDefName := fileData.Model.ModelDefinition
			if err != nil {
				return "", &datamodel.Model{}, err
			}
			uploadedModel = datamodel.Model{
				ID:              fileData.Model.Id,
				Visibility:      datamodel.ModelVisibility(visibility),
				Description:     description,
				ModelDefinition: modelDefName,
				Instances: []datamodel.ModelInstance{{
					ModelDefinition: modelDefName,
					State:           datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
					ID:              "latest",
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

func savePredictInputs(stream modelPB.ModelService_TriggerModelInstanceBinaryFileUploadServer) (imageBytes [][]byte, modelId string, instanceId string, err error) {
	var firstChunk = true
	var fileData *modelPB.TriggerModelInstanceBinaryFileUploadRequest

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
			modelId, instanceId, err = getModelInstanceID(fileData.Name) // format "models/{model}/instances/{instance}"
			if err != nil {
				return [][]byte{}, "", "", err
			}

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
	return imageBytes, modelId, instanceId, nil
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

func (s *handler) Liveness(ctx context.Context, pb *modelPB.LivenessRequest) (*modelPB.LivenessResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.LivenessResponse{HealthCheckResponse: &modelPB.HealthCheckResponse{Status: modelPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	return &modelPB.LivenessResponse{HealthCheckResponse: &modelPB.HealthCheckResponse{Status: modelPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (s *handler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.ReadinessResponse{HealthCheckResponse: &modelPB.HealthCheckResponse{Status: modelPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	return &modelPB.ReadinessResponse{HealthCheckResponse: &modelPB.HealthCheckResponse{Status: modelPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

///////////////////////////////////////////////////////
///////////////////   MODEL HANDLERS //////////////////
func HandleCreateModelByMultiPartFormData(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		owner := r.Header.Get("owner")
		if owner == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}

		if strings.Contains(owner, "..") {
			makeJsonResponse(w, 422, "owner error", "The user name should not contain special characters")
			return
		}

		modelName := r.FormValue("name")
		if modelName == "" {
			makeJsonResponse(w, 400, "Missing parameter", "Model name need to be specified")
			return
		}

		modeId, err := getID(modelName)
		if err != nil {
			makeJsonResponse(w, 400, "Missing parameter", "Model name is invalid")
			return
		}

		modelDefinitionName := r.FormValue("model_definition_name")
		if modelDefinitionName == "" {
			makeJsonResponse(w, 400, "Missing parameter", "modelDefinitionName need to be specified")
			return
		}
		modelDefinitionId, err := getDefinitionUID(modelDefinitionName)
		if err != nil {
			makeJsonResponse(w, 400, "Invalid parameter", err.Error())
			return
		}

		viz := r.FormValue("visibility")
		var visibility modelPB.Model_Visibility
		if viz != "" {
			if util.Visibility[viz] == modelPB.Model_VISIBILITY_UNSPECIFIED {
				makeJsonResponse(w, 400, "Invalid parameter", "Visibility is invalid")
				return
			} else {
				visibility = util.Visibility[viz]
			}
		} else {
			visibility = modelPB.Model_VISIBILITY_PRIVATE
		}

		err = r.ParseMultipartForm(4 << 20)
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
		rdid, _ := uuid.NewV4()
		tmpFile := path.Join("/tmp", rdid.String())
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

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		modelService := service.NewService(modelRepository, tritonService)

		_, err = modelRepository.GetModelDefinition(modelDefinitionId)
		if err != nil {
			makeJsonResponse(w, 400, "Parameter invalid", "ModelDefinitionId not found")
			return
		}

		var uploadedModel = datamodel.Model{
			Instances: []datamodel.ModelInstance{{
				State:           datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
				ID:              "latest",
				ModelDefinition: modelDefinitionName,
			}},
			ID:              modeId,
			ModelDefinition: modelDefinitionName,
			Owner:           owner,
			Visibility:      datamodel.ModelVisibility(visibility),
			Description:     r.FormValue("description"),
		}

		_, err = modelService.GetModelById(owner, uploadedModel.ID)
		if err == nil {
			makeJsonResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
			return
		}

		readmeFilePath, err := unzip(tmpFile, configs.Config.TritonServer.ModelStore, owner, &uploadedModel)
		_ = os.Remove(tmpFile) // remove uploaded temporary zip file
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJsonResponse(w, 400, "Add Model Error", err.Error())
			return
		}
		if _, err := os.Stat(readmeFilePath); err == nil {
			modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
			if err != nil {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
				makeJsonResponse(w, 400, "Add Model Error", err.Error())
				return
			}
			if modelMeta.Task == "" {
				uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
			} else {
				if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
					uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(val)
				} else {
					util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
					makeJsonResponse(w, 400, "Add Model Error", "README.md contains unsupported task")
					return
				}
			}
		} else {
			uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		}

		dbModel, err := modelService.CreateModel(owner, &uploadedModel)
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJsonResponse(w, 500, "Add Model Error", err.Error())
			return
		}
		pbModel := DBModelToPBModel(&dbModel)

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(201)

		m := jsonpb.Marshaler{OrigName: true, EnumsAsInts: false, EmitDefaults: true}
		var buffer bytes.Buffer
		err = m.Marshal(&buffer, &modelPB.CreateModelResponse{Model: pbModel})
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
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
	owner, err := getOwner(stream.Context())
	if err != nil {
		return err
	}
	tmpFile, uploadedModel, err := saveFile(stream)
	if err != nil {
		return makeError(codes.InvalidArgument, "Save File Error", err.Error())
	}
	_, err = s.service.GetModelById(owner, uploadedModel.ID)
	if err == nil {
		return makeError(codes.AlreadyExists, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
	}

	uploadedModel.Owner = owner

	// extract zip file from tmp to models directory
	readmeFilePath, err := unzip(tmpFile, configs.Config.TritonServer.ModelStore, owner, uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return makeError(codes.InvalidArgument, "Save File Error", err.Error())
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			return makeError(codes.InvalidArgument, "Add Model Error", err.Error())
		}
		if modelMeta.Task == "" {
			uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		} else {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(val)
			} else {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
				return makeError(codes.InvalidArgument, "Add Model Error", "README.md contains unsupported task")
			}
		}
	} else {
		uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
	}

	dbModel, err := s.service.CreateModel(owner, uploadedModel)
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return err
	}
	pbModel := DBModelToPBModel(&dbModel)
	err = stream.SendAndClose(&modelPB.CreateModelBinaryFileUploadResponse{Model: pbModel})
	if err != nil {
		return makeError(codes.Internal, "Add Model Error", err.Error())
	}

	return
}

func (s *handler) CreateModel(ctx context.Context, req *modelPB.CreateModelRequest) (*modelPB.CreateModelResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkOutputOnlyFields(req.Model); err != nil {
		return &modelPB.CreateModelResponse{}, err
	}

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkRequiredFields(req.Model); err != nil {
		return &modelPB.CreateModelResponse{}, err
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkResourceID(req.Model.GetId()); err != nil {
		return &modelPB.CreateModelResponse{}, err
	}

	modelDefinitionId, err := getDefinitionUID(req.Model.ModelDefinition)
	if err != nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	_, err = s.service.GetModelDefinition(modelDefinitionId)
	if err != nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}

	_, err = s.service.GetModelById(owner, req.Model.Id)
	if err == nil {
		return &modelPB.CreateModelResponse{}, fmt.Errorf("The model %v already existed", req.Model.Id)
	}

	if req.Model.Configuration == nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", "Missing Configuration")
	}

	b, err := req.Model.Configuration.Specification.MarshalJSON()
	if err != nil {
		return &modelPB.CreateModelResponse{}, err
	}
	var github datamodel.ModelInstanceConfiguration
	err = json.Unmarshal(b, &github)
	if err != nil {
		return &modelPB.CreateModelResponse{}, err
	}

	if github.Repo == "" || github.Tag == "" || !util.IsGitHubURL(github.Repo) {
		return &modelPB.CreateModelResponse{}, makeError(codes.FailedPrecondition, "Add Model Error", "Invalid GitHub URL")
	}

	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
	err = util.GitHubClone(modelSrcDir, github)
	if err != nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	githubInfo, err := util.GetGitHubRepoInfo(github.Repo)
	if err != nil {
		return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", err.Error())
	}
	visibility := util.Visibility[githubInfo.Visibility]
	if req.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	} else if req.Model.Visibility == modelPB.Model_VISIBILITY_PRIVATE {
		visibility = modelPB.Model_VISIBILITY_PRIVATE
	}

	githubConfigObj, _ := json.Marshal(github)
	githubModelConfig, _ := json.Marshal(datamodel.ModelConfiguration{
		Repo: github.Repo,
	})

	githubModel := datamodel.Model{
		ID:              req.Model.Id,
		ModelDefinition: req.Model.ModelDefinition,
		Owner:           owner,
		Visibility:      datamodel.ModelVisibility(visibility),
		Description:     githubInfo.Description,
		Configuration: datamodel.Spec{
			DocumentationUrl: "",
			Specification:    githubModelConfig,
		},
		Instances: []datamodel.ModelInstance{{
			ID:              github.Tag,
			ModelDefinition: req.Model.ModelDefinition,
			State:           datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			Configuration: datamodel.Spec{
				DocumentationUrl: "",
				Specification:    githubConfigObj,
			},
		}},
	}

	readmeFilePath, err := updateModelPath(modelSrcDir, configs.Config.TritonServer.ModelStore, owner, &githubModel)
	_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, githubModel.ID, githubModel.Instances[0].ID)
		return &modelPB.CreateModelResponse{}, err
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil || modelMeta.Task == "" {
			util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, githubModel.ID, githubModel.Instances[0].ID)
			return &modelPB.CreateModelResponse{}, err
		}
		if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			githubModel.Instances[0].Task = datamodel.ModelInstanceTask(val)
		} else {
			if modelMeta.Task != "" {
				util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, githubModel.ID, githubModel.Instances[0].ID)
				return &modelPB.CreateModelResponse{}, makeError(codes.InvalidArgument, "Add Model Error", "README.md contains unsupported task")
			} else {
				githubModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
			}
		}
	} else {
		githubModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
	}
	dbModel, err := s.service.CreateModel(owner, &githubModel)
	if err != nil {
		util.RemoveModelRepository(configs.Config.TritonServer.ModelStore, owner, githubModel.ID, githubModel.Instances[0].ID)
		return &modelPB.CreateModelResponse{}, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	pbModel := DBModelToPBModel(&dbModel)

	return &modelPB.CreateModelResponse{Model: pbModel}, nil
}

func (s *handler) ListModel(ctx context.Context, req *modelPB.ListModelRequest) (*modelPB.ListModelResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.ListModelResponse{}, err
	}

	dbModels, nextPageToken, totalSize, err := s.service.ListModel(owner, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelResponse{}, err
	}

	pbModels := []*modelPB.Model{}
	for _, dbModel := range dbModels {
		pbModels = append(pbModels, DBModelToPBModel(&dbModel))
	}

	resp := modelPB.ListModelResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (s *handler) GetModel(ctx context.Context, req *modelPB.GetModelRequest) (*modelPB.GetModelResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	id, err := getID(req.Name)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	dbModel, err := s.service.GetModelById(owner, id)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&dbModel)
	return &modelPB.GetModelResponse{Model: pbModel}, err
}

func (s *handler) DeleteModel(ctx context.Context, req *modelPB.DeleteModelRequest) (*modelPB.DeleteModelResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.DeleteModelResponse{}, err
	}
	id, err := getID(req.Name)
	if err != nil {
		return &modelPB.DeleteModelResponse{}, err
	}
	return &modelPB.DeleteModelResponse{}, s.service.DeleteModel(owner, id)
}

func (s *handler) RenameModel(ctx context.Context, req *modelPB.RenameModelRequest) (*modelPB.RenameModelResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	id, err := getID(req.Name)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	dbModel, err := s.service.RenameModel(owner, id, req.NewModelId)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&dbModel)
	return &modelPB.RenameModelResponse{Model: pbModel}, nil
}

///////////////////////////////////////////////////////
/////////////   MODEL INSTANCE HANDLERS ///////////////
func (s *handler) GetModelInstance(ctx context.Context, req *modelPB.GetModelInstanceRequest) (*modelPB.GetModelInstanceResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	modelId, instanceId, err := getModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	dbModel, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	dbModelInstance, err := s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	pbModelInstance := DBModelInstanceToPBModelInstance(modelId, &dbModelInstance)
	return &modelPB.GetModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (s *handler) ListModelInstance(ctx context.Context, req *modelPB.ListModelInstanceRequest) (*modelPB.ListModelInstanceResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	modelId, err := getID(req.Parent)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}
	modelInDB, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	dbModelInstances, nextPageToken, totalSize, err := s.service.ListModelInstance(modelInDB.UID, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	pbInstances := []*modelPB.ModelInstance{}
	for _, dbModelInstance := range dbModelInstances {
		pbInstances = append(pbInstances, DBModelInstanceToPBModelInstance(modelId, &dbModelInstance))
	}

	resp := modelPB.ListModelInstanceResponse{
		Instances:     pbInstances,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (s *handler) DeployModelInstance(ctx context.Context, req *modelPB.DeployModelInstanceRequest) (*modelPB.DeployModelInstanceResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	modelId, instanceId, err := getModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	dbModel, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	dbModelInstance, err := s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	err = s.service.DeployModelInstance(dbModelInstance.UID)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	dbModelInstance, err = s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}
	pbModelInstance := DBModelInstanceToPBModelInstance(modelId, &dbModelInstance)

	return &modelPB.DeployModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (s *handler) UndeployModelInstance(ctx context.Context, req *modelPB.UndeployModelInstanceRequest) (*modelPB.UndeployModelInstanceResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	modelId, instanceId, err := getModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModel, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModelInstance, err := s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	err = s.service.UndeployModelInstance(dbModelInstance.UID)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModelInstance, err = s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}
	pbModelInstance := DBModelInstanceToPBModelInstance(modelId, &dbModelInstance)

	return &modelPB.UndeployModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (s *handler) TriggerModelInstanceBinaryFileUpload(stream modelPB.ModelService_TriggerModelInstanceBinaryFileUploadServer) error {
	if !s.triton.IsTritonServerReady() {
		return makeError(503, "TriggerModelInstanceBinaryFileUpload", "Triton Server not ready yet")
	}

	owner, err := getOwner(stream.Context())
	if err != nil {
		return err
	}

	imageBytes, modelId, instanceId, err := savePredictInputs(stream)
	if err != nil {
		return makeError(500, "TriggerModelInstanceBinaryFileUpload", "Could not save the file")
	}

	modelInDB, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return makeError(404, "TriggerModelInstanceBinaryFileUpload", fmt.Sprintf("The model %v do not exist", modelId))
	}
	modelInstanceInDB, err := s.service.GetModelInstance(modelInDB.UID, instanceId)
	if err != nil {
		return makeError(404, "TriggerModelInstanceBinaryFileUpload", fmt.Sprintf("The model instance %v do not exist", instanceId))
	}
	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := s.service.ModelInfer(modelInstanceInDB.UID, imageBytes, task)
	if err != nil {
		return err
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return makeError(500, "TriggerModelInstanceBinaryFileUpload", err.Error())
		}
	case modelPB.ModelInstance_TASK_DETECTION:
		b, err = json.Marshal(response.(*modelPB.DetectionOutputs))
		if err != nil {
			return makeError(500, "TriggerModelInstanceBinaryFileUpload", err.Error())
		}
	default:
		b, err = json.Marshal(response.(*inferenceserver.ModelInferResponse))
		if err != nil {
			return makeError(500, "TriggerModelInstanceBinaryFileUpload", err.Error())
		}
	}
	err = protojson.Unmarshal(b, data)
	if err != nil {
		return makeError(500, "TriggerModelInstanceBinaryFileUpload", err.Error())
	}
	err = stream.SendAndClose(&modelPB.TriggerModelInstanceBinaryFileUploadResponse{Output: data})
	return err
}

func (s *handler) TriggerModelInstance(ctx context.Context, req *modelPB.TriggerModelInstanceRequest) (*modelPB.TriggerModelInstanceResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	modelId, modelInstanceId, err := getModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	modelInDB, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.NotFound, "TriggerModelInstance", fmt.Sprintf("The model instance named %v not found in server", req.Name))
	}

	modelInstanceInDB, err := s.service.GetModelInstance(modelInDB.UID, modelInstanceId)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.NotFound, "TriggerModelInstance", fmt.Sprintf("The model %v  with instance %v not found in server", modelId, modelInstanceId))
	}

	imgsBytes, _, err := ParseImageRequestInputsToBytes(req)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.InvalidArgument, "TriggerModelInstance", err.Error())
	}
	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := s.service.ModelInfer(modelInstanceInDB.UID, imgsBytes, task)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.InvalidArgument, "TriggerModelInstance", err.Error())
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.Internal, "TriggerModelInstance", err.Error())
		}
	case modelPB.ModelInstance_TASK_DETECTION:
		b, err = json.Marshal(response.(*modelPB.DetectionOutputs))
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.Internal, "TriggerModelInstance", err.Error())
		}
	default:
		b, err = json.Marshal(response.(*inferenceserver.ModelInferResponse))
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.Internal, "TriggerModelInstance", err.Error())
		}
	}
	err = protojson.Unmarshal(b, data)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, makeError(codes.Internal, "TriggerModelInstance", err.Error())
	}

	return &modelPB.TriggerModelInstanceResponse{Output: data}, nil
}

func HandleTriggerModelInstanceByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		owner := r.Header.Get("owner")

		if owner == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}

		instanceName := pathParams["name"]
		if instanceName == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter mode name not found")
			return
		}

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		modelService := service.NewService(modelRepository, tritonService)

		modelId, instanceId, err := getModelInstanceID(instanceName)
		if err != nil {
			makeJsonResponse(w, 400, "Parameter invalid", "Required parameter instance_name is invalid")
			return
		}

		modelInDB, err := modelService.GetModelById(owner, modelId)
		if err != nil {
			makeJsonResponse(w, 404, "Model not found", "The model not found in server")
			return
		}

		modelInstanceInDB, err := modelService.GetModelInstance(modelInDB.UID, instanceId)
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
		response, err := modelService.ModelInfer(modelInstanceInDB.UID, imgsBytes, task)
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
		res, err := json.Marshal(&modelPB.TriggerModelInstanceBinaryFileUploadResponse{Output: data})
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

func (s *handler) GetModelInstanceCard(ctx context.Context, req *modelPB.GetModelInstanceCardRequest) (*modelPB.GetModelInstanceCardResponse, error) {
	owner, err := getOwner(ctx)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	modelId, instanceId, err := getModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	dbModel, err := s.service.GetModelById(owner, modelId)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	_, err = s.service.GetModelInstance(dbModel.UID, instanceId)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md#%v", configs.Config.TritonServer.ModelStore, owner, modelId, instanceId)
	stat, err := os.Stat(readmeFilePath)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	f, _ := os.Open(readmeFilePath)
	reader := bufio.NewReader(f)
	content, _ := ioutil.ReadAll(reader)
	// Encode as base64.
	encoded := base64.StdEncoding.EncodeToString(content)

	return &modelPB.GetModelInstanceCardResponse{Readme: &modelPB.ModelInstanceCard{
		Name:     req.Name,
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Contents: []byte(encoded),
	}}, nil
}

///////////////////////////////////////////////////////
/////////////   MODEL DEFINITION HANDLERS /////////////
func (s *handler) GetModelDefinition(ctx context.Context, req *modelPB.GetModelDefinitionRequest) (*modelPB.GetModelDefinitionResponse, error) {
	definitionId, err := getDefinitionUID(req.Name)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	dbModelDefinition, err := s.service.GetModelDefinition(definitionId)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	pbModelInstance := DBModelDefinitionToPBModelDefinition(&dbModelDefinition)
	return &modelPB.GetModelDefinitionResponse{ModelDefinition: pbModelInstance}, nil
}

func (s *handler) ListModelDefinition(ctx context.Context, req *modelPB.ListModelDefinitionRequest) (*modelPB.ListModelDefinitionResponse, error) {

	dbModelDefinitions, nextPageToken, totalSize, err := s.service.ListModelDefinition(req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelDefinitionResponse{}, err
	}

	pbDefinitions := []*modelPB.ModelDefinition{}
	for _, dbModelDefinition := range dbModelDefinitions {
		pbDefinitions = append(pbDefinitions, DBModelDefinitionToPBModelDefinition(&dbModelDefinition))
	}

	resp := modelPB.ListModelDefinitionResponse{
		ModelDefinitions: pbDefinitions,
		NextPageToken:    nextPageToken,
		TotalSize:        totalSize,
	}

	return &resp, nil
}
