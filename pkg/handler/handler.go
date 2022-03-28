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
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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

func unzip(filePath string, dstDir string, namespace string, uploadedModel *datamodel.Model) bool {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return false
	}
	defer archive.Close()

	var createdTModels []datamodel.TModel
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
			return false
		}
		if f.FileInfo().IsDir() {
			dirName := f.Name
			if string(dirName[len(dirName)-1]) == "/" {
				dirName = dirName[:len(dirName)-1]
			}
			if !strings.Contains(dirName, "/") { // top directory model
				currentOldModelName = dirName
				dirName = fmt.Sprintf("%v#%v#%v#%v", namespace, uploadedModel.Name, dirName, uploadedModel.Versions[0].Version)
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
						createdTModels = append(createdTModels, datamodel.TModel{
							Name:    currentNewModelName, // Triton model name
							Status:  modelPB.ModelVersion_STATUS_OFFLINE.String(),
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
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", namespace, uploadedModel.Name, subStrs[0], uploadedModel.Versions[0].Version)
		newModelName := subStrs[0]
		filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return false
		}
		fileInArchive, err := f.Open()
		if err != nil {
			return false
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return false
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
				return false
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = updateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return false
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
	return true
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
			uploadedModel = datamodel.Model{
				Name:     fileData.Name,
				Task:     uint64(fileData.Task),
				Versions: []datamodel.Version{},
			}
			uploadedModel.Versions = append(uploadedModel.Versions, datamodel.Version{
				Description: fileData.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      modelPB.ModelVersion_STATUS_OFFLINE.String(),
				Version:     1,
			})
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

func savePredictInput(stream modelPB.ModelService_TriggerModelBinaryFileUploadServer) (imageByte []byte, modelId string, version uint64, err error) {
	var firstChunk = true
	var fileData *modelPB.TriggerModelBinaryFileUploadRequest

	var fileContent []byte

	for {
		fileData, err = stream.Recv() //ignoring the data  TO-Do save files received
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return []byte{}, "", 0, err
		}

		if firstChunk { //first chunk contains file name
			modelId = fileData.Name
			version = fileData.Version

			firstChunk = false
		}
		fileContent = append(fileContent, fileData.Bytes...)
	}
	return fileContent, modelId, version, nil
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

		var task = 0
		sTask := r.FormValue("task")
		if val, ok := util.Tasks[sTask]; ok {
			task = val
		} else {
			if sTask != "" {
				makeJsonResponse(w, 400, "Parameter Error", "Wrong CV Task value")
				return
			}
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

		var uploadedModel = datamodel.Model{
			Versions: []datamodel.Version{},
			Name:     modelName,
			Task:     uint64(task),
		}
		uploadedModel.Versions = append(uploadedModel.Versions, datamodel.Version{
			Description: r.FormValue("description"),
			Status:      modelPB.ModelVersion_STATUS_OFFLINE.String(),
			Version:     1,
		})
		uploadedModel.Namespace = username

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		modelService := service.NewService(modelRepository, tritonService)

		modelInDB, err := modelService.GetModelByName(username, uploadedModel.Name)
		if err == nil {
			latestVersion, err := modelService.GetModelVersionLatest(modelInDB.Id)
			if err == nil {
				uploadedModel.Versions[0].Version = latestVersion.Version + 1
			}
			if modelInDB.Task != uploadedModel.Task {
				makeJsonResponse(w, 400, "Invalid task value", fmt.Sprintf("The model have task %v which need to be consistency", modelInDB.Task))
				return
			}
		}
		isOk := unzip(tmpFile, configs.Config.TritonServer.ModelStore, username, &uploadedModel)
		_ = os.Remove(tmpFile) // remove uploaded temporary zip file
		if !isOk {
			makeJsonResponse(w, 400, "Add Model Error", "Could not extract zip file")
			return
		}

		resModel, err := modelService.CreateModelBinaryFileUpload(username, &uploadedModel)
		if err != nil {
			makeJsonResponse(w, 500, "Add Model Error", err.Error())
			return
		}
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		m := jsonpb.Marshaler{OrigName: true, EnumsAsInts: false, EmitDefaults: true}
		var buffer bytes.Buffer
		err = m.Marshal(&buffer, &modelPB.CreateModelBinaryFileUploadResponse{Model: resModel})
		if err != nil {
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
	modelInDB, err := s.service.GetModelByName(username, uploadedModel.Name)
	if err == nil {
		latestVersion, err := s.service.GetModelVersionLatest(modelInDB.Id)
		if err == nil {
			uploadedModel.Versions[0].Version = latestVersion.Version + 1
		}

		if modelInDB.Task != uploadedModel.Task {
			return makeError(codes.InvalidArgument, "Invalid task value", fmt.Sprintf("The model have task %v which need to be consistency", modelInDB.Task))
		}
	}

	uploadedModel.Namespace = username
	// extract zip file from tmp to models directory
	isOk := unzip(tmpFile, configs.Config.TritonServer.ModelStore, username, uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if !isOk {
		return makeError(codes.InvalidArgument, "Save File Error", "Could not extract zip file")
	}
	resModel, err := s.service.CreateModelBinaryFileUpload(username, uploadedModel)
	if err != nil {
		return err
	}
	err = stream.SendAndClose(&modelPB.CreateModelBinaryFileUploadResponse{Model: resModel})
	if err != nil {
		return makeError(codes.Internal, "Add Model Error", err.Error())
	}

	return
}

func (s *handler) UpdateModelVersion(ctx context.Context, in *modelPB.UpdateModelVersionRequest) (*modelPB.UpdateModelVersionResponse, error) {
	if !s.triton.IsTritonServerReady() {
		return &modelPB.UpdateModelVersionResponse{}, makeError(503, "LoadModel Error", "Triton Server not ready yet")
	}

	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.UpdateModelVersionResponse{}, err
	}
	modelVersion, err := s.service.UpdateModelVersion(username, in)
	return &modelPB.UpdateModelVersionResponse{ModelVersion: modelVersion}, err
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

	modelInDB, err := s.service.GetModelByName(username, in.Name)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.NotFound, "PredictModel", fmt.Sprintf("The model named %v not found in server", in.Name))
	}

	_, err = s.service.GetModelVersion(modelInDB.Id, in.Version)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.NotFound, "PredictModel", fmt.Sprintf("The model %v  with version %v not found in server", in.Name, in.Version))
	}

	imgsBytes, _, err := ParseImageRequestInputsToBytes(in)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.InvalidArgument, "PredictModel", err.Error())
	}
	task := modelPB.Model_Task(modelInDB.Task)
	response, err := s.service.ModelInfer(username, in.Name, in.Version, imgsBytes, task)
	if err != nil {
		return &modelPB.TriggerModelResponse{}, makeError(codes.InvalidArgument, "PredictModel", err.Error())
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return &modelPB.TriggerModelResponse{}, makeError(codes.Internal, "PredictModel", err.Error())
		}
	case modelPB.Model_TASK_DETECTION:
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

	imageByte, modelName, version, err := savePredictInput(stream)
	if err != nil {
		return makeError(500, "PredictModel", "Could not save the file")
	}

	modelInDB, err := s.service.GetModelByName(username, modelName)
	if err != nil {
		return makeError(404, "PredictModel", fmt.Sprintf("The model %v do not exist", modelName))
	}
	task := modelPB.Model_Task(modelInDB.Task)

	response, err := s.service.ModelInfer(username, modelName, version, [][]byte{imageByte}, task)

	if err != nil {
		return err
	}

	var data = &structpb.Struct{}
	var b []byte
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION:
		b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}
	case modelPB.Model_TASK_DETECTION:
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
		modelName := pathParams["name"]

		if username == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}
		if modelName == "" {
			makeJsonResponse(w, 422, "Required parameter missing", "Required parameter mode name not found")
			return
		}

		modelVersion, err := strconv.ParseInt(pathParams["version"], 10, 32)

		if err != nil {
			makeJsonResponse(w, 400, "Wrong parameter type", "Version should be a number greater than 0")
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

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJsonResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}

		imgsBytes, _, err := parseImageFormDataInputsToBytes(r)
		if err != nil {
			makeJsonResponse(w, 400, "File Input Error", err.Error())
			return
		}

		task := modelPB.Model_Task(modelInDB.Task)
		response, err := modelService.ModelInfer(username, modelName, uint64(modelVersion), imgsBytes, task)
		if err != nil {
			makeJsonResponse(w, 500, "Error Predict Model", err.Error())
			return
		}
		var data = &structpb.Struct{}
		var b []byte
		switch task {
		case modelPB.Model_TASK_CLASSIFICATION:
			b, err = json.Marshal(response.(*modelPB.ClassificationOutputs))
			if err != nil {
				makeJsonResponse(w, 500, "Error Predict Model", err.Error())
				return
			}
		case modelPB.Model_TASK_DETECTION:
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

func (s *handler) DeleteModelVersion(ctx context.Context, in *modelPB.DeleteModelVersionRequest) (*modelPB.DeleteModelVersionResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &modelPB.DeleteModelVersionResponse{}, err
	}
	return &modelPB.DeleteModelVersionResponse{}, s.service.DeleteModelVersion(username, in.Name, in.Version)
}
