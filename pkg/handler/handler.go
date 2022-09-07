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

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/external"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	database "github.com/instill-ai/model-backend/internal/db"
	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"Id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"Name", "Uid", "Visibility", "Owner", "CreateTime", "UpdateTime"}

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
	datamodel.InitJSONSchema()
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

func isEnsembleConfig(configPath string) bool {
	fileData, _ := ioutil.ReadFile(configPath)
	fileString := string(fileData)
	return strings.Contains(fileString, "platform: \"ensemble\"")
}

func unzip(filePath string, dstDir string, owner string, uploadedModel *datamodel.Model) (string, string, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return "", "", err
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
			return "", "", fmt.Errorf("invalid file path")
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
			if err := util.ValidateFilePath(filePath); err != nil {
				return "", "", err
			}
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return "", "", err
			}
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
		if err := util.ValidateFilePath(filePath); err != nil {
			return "", "", err
		}
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", "", err
		}
		fileInArchive, err := f.Open()
		if err != nil {
			return "", "", err
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return "", "", err
		}

		dstFile.Close()
		fileInArchive.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(filePath)
		if fileExtension == ".pbtxt" {
			if isEnsembleConfig(filePath) {
				ensembleFilePath = filePath
			}
			err = util.UpdateConfigModelName(filePath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = util.UpdateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	uploadedModel.Instances[0].TritonModels = createdTModels
	return readmeFilePath, ensembleFilePath, nil
}

// modelDir and dstDir are absolute path
func updateModelPath(modelDir string, dstDir string, owner string, modelID string, modelInstance *datamodel.ModelInstance) (string, string, error) {
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
		return "", "", err
	}
	modelRootDir := strings.Join([]string{dstDir, owner}, "/")
	err = os.MkdirAll(modelRootDir, os.ModePerm)
	if err != nil {
		return "", "", err
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
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", owner, modelID, oldModelName, modelInstance.ID)
		var filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))

		if f.fInfo.IsDir() { // create new folder
			err = os.MkdirAll(filePath, os.ModePerm)

			if err != nil {
				return "", "", err
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
			return "", "", err
		}
		srcFile, err := os.Open(f.path)
		if err != nil {
			return "", "", err
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return "", "", err
		}
		dstFile.Close()
		srcFile.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(filePath)
		if fileExtension == ".pbtxt" {
			if isEnsembleConfig(filePath) {
				ensembleFilePath = filePath
			}
			err = util.UpdateConfigModelName(filePath, oldModelName, subStrs[0])
			if err != nil {
				return "", "", err
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = util.UpdateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	modelInstance.TritonModels = createdTModels
	return readmeFilePath, ensembleFilePath, nil
}

func saveFile(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (outFile string, modelInfo *datamodel.Model, modelDefinitionID string, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateModelBinaryFileUploadRequest

	var tmpFile string

	var uploadedModel datamodel.Model
	for {
		fileData, err = stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { //first chunk contains file name
			if fileData.Model == nil {
				return "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
			}

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
			modelDefinitionID, err = resource.GetDefinitionID(modelDefName)
			if err != nil {
				return "", &datamodel.Model{}, "", err
			}
			uploadedModel = datamodel.Model{
				ID:            fileData.Model.Id,
				Visibility:    datamodel.ModelVisibility(visibility),
				Description:   description,
				Configuration: datatypes.JSON{},
				Instances: []datamodel.ModelInstance{{
					State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
					ID:    "latest",
				}},
			}
			if err != nil {
				return "", &datamodel.Model{}, "", err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = writeToFp(fp, fileData.Content)
		if err != nil {
			return "", &datamodel.Model{}, "", err
		}
	}
	return tmpFile, &uploadedModel, modelDefinitionID, nil
}

func savePredictInputsTriggerMode(stream modelPB.ModelService_TriggerModelInstanceBinaryFileUploadServer) (imageBytes [][]byte, modelID string, instanceID string, err error) {
	var firstChunk = true
	var fileData *modelPB.TriggerModelInstanceBinaryFileUploadRequest

	var allContentFiles []byte
	var fileLengths []uint64
	for {
		fileData, err = stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return [][]byte{}, "", "", err
		}

		if firstChunk { //first chunk contains model instance name
			firstChunk = false
			modelID, instanceID, err = resource.GetModelInstanceID(fileData.Name) // format "models/{model}/instances/{instance}"
			if err != nil {
				return [][]byte{}, "", "", err
			}
			fileLengths = fileData.FileLengths
			if len(fileLengths) == 0 {
				return [][]byte{}, "", "", fmt.Errorf("wrong parameter length of files")
			}
			allContentFiles = append(allContentFiles, fileData.Content...)
		} else {
			allContentFiles = append(allContentFiles, fileData.Content...)
		}
	}

	imageBytes = make([][]byte, len(fileLengths))
	start := uint64(0)
	for i := 0; i < len(fileLengths); i++ {
		imageBytes[i] = allContentFiles[start : start+fileLengths[i]]
		start += fileLengths[i]
	}

	return imageBytes, modelID, instanceID, nil
}

func savePredictInputsTestMode(stream modelPB.ModelService_TestModelInstanceBinaryFileUploadServer) (imageBytes [][]byte, modelID string, instanceID string, err error) {
	var firstChunk = true
	var fileData *modelPB.TestModelInstanceBinaryFileUploadRequest

	var allContentFiles []byte
	var fileLengths []uint64
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
			modelID, instanceID, err = resource.GetModelInstanceID(fileData.Name) // format "models/{model}/instances/{instance}"
			if err != nil {
				return [][]byte{}, "", "", err
			}

			fileLengths = fileData.FileLengths
			if len(fileLengths) == 0 {
				return [][]byte{}, "", "", fmt.Errorf("wrong parameter length of files")
			}
			allContentFiles = append(allContentFiles, fileData.Content...)
			firstChunk = false
		}
		allContentFiles = append(allContentFiles, fileData.Content...)
	}

	if len(fileLengths) == 0 {
		return [][]byte{}, "", "", fmt.Errorf("wrong parameter length of files")
	}
	start := uint64(0)
	for i := 0; i < len(fileLengths); i++ {
		imageBytes[i] = allContentFiles[start : start+fileLengths[i]]
		start += fileLengths[i]
	}
	return imageBytes, modelID, instanceID, nil
}

func makeJSONResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

func (h *handler) Liveness(ctx context.Context, pb *modelPB.LivenessRequest) (*modelPB.LivenessResponse, error) {
	if !h.triton.IsTritonServerReady() {
		return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (h *handler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !h.triton.IsTritonServerReady() {
		return &modelPB.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	return &modelPB.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

// HandleCreateModelByMultiPartFormData is a custom handler
func HandleCreateModelByMultiPartFormData(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	logger, _ := logger.GetZapLogger()

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		owner, err := resource.GetOwnerFromHeader(r)
		if err != nil || owner == "" {
			makeJSONResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}

		if strings.Contains(owner, "..") {
			makeJSONResponse(w, 422, "owner error", "The user name should not contain special characters")
			return
		}

		modelID := r.FormValue("id")
		if modelID == "" {
			makeJSONResponse(w, 400, "Missing parameter", "Model Id need to be specified")
			return
		}

		modelDefinitionName := r.FormValue("model_definition")
		if modelDefinitionName == "" {
			makeJSONResponse(w, 400, "Missing parameter", "modelDefinitionName need to be specified")
			return
		}
		modelDefinitionID, err := resource.GetDefinitionID(modelDefinitionName)
		if err != nil {
			makeJSONResponse(w, 400, "Invalid parameter", err.Error())
			return
		}

		viz := r.FormValue("visibility")
		var visibility modelPB.Model_Visibility
		if viz != "" {
			if util.Visibility[viz] == modelPB.Model_VISIBILITY_UNSPECIFIED {
				makeJSONResponse(w, 400, "Invalid parameter", "Visibility is invalid")
				return
			} else {
				visibility = util.Visibility[viz]
			}
		} else {
			visibility = modelPB.Model_VISIBILITY_PRIVATE
		}

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}
		file, fileHeader, err := r.FormFile("content")
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", "Error while reading file from request")
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
			makeJSONResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		rdid, _ := uuid.NewV4()
		tmpFile := path.Join("/tmp", rdid.String())
		fp, err := os.Create(tmpFile)
		if err != nil {
			makeJSONResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		err = writeToFp(fp, buf.Bytes())
		if err != nil {
			makeJSONResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		defer tritonService.Close()
		pipelineServiceClient, pipelineServiceClientConn := external.InitPipelineServiceClient()
		defer pipelineServiceClientConn.Close()
		redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
		defer redisClient.Close()

		modelService := service.NewService(modelRepository, tritonService, pipelineServiceClient, redisClient)

		// validate model configuration
		localModelDefinition, err := modelRepository.GetModelDefinition(modelDefinitionID)
		if err != nil {
			makeJSONResponse(w, 400, "Parameter invalid", "ModelDefinitionId not found")
			return
		}
		rs := &jsonschema.Schema{}
		if err := json.Unmarshal([]byte(localModelDefinition.ModelSpec.String()), rs); err != nil {
			makeJSONResponse(w, 500, "Add Model Error", "Could not get model definition")
			return
		}
		modelConfiguration := datamodel.LocalModelConfiguration{
			Content: fileHeader.Filename,
		}

		if err := datamodel.ValidateJSONSchema(rs, modelConfiguration, true); err != nil {
			makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model configuration is invalid %v", err.Error()))
			return
		}
		bModelConfig, _ := json.Marshal(modelConfiguration)
		var uploadedModel = datamodel.Model{
			Instances: []datamodel.ModelInstance{{
				State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
				ID:            "latest",
				Configuration: bModelConfig,
			}},
			ID:                 modelID,
			ModelDefinitionUid: localModelDefinition.UID,
			Owner:              owner,
			Visibility:         datamodel.ModelVisibility(visibility),
			Description:        r.FormValue("description"),
			Configuration:      bModelConfig,
		}

		// Validate ModelDefinition JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, DBModelToPBModel(&localModelDefinition, &uploadedModel), true); err != nil {
			makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model configuration is invalid %v", err.Error()))
			return
		}

		_, err = modelService.GetModelById(owner, uploadedModel.ID, modelPB.View_VIEW_FULL)
		if err == nil {
			makeJSONResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
			return
		}

		readmeFilePath, ensembleFilePath, err := unzip(tmpFile, config.Config.TritonServer.ModelStore, owner, &uploadedModel)
		_ = os.Remove(tmpFile) // remove uploaded temporary zip file
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJSONResponse(w, 400, "Add Model Error", err.Error())
			return
		}
		if _, err := os.Stat(readmeFilePath); err == nil {
			modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
			if err != nil {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
				makeJSONResponse(w, 400, "Add Model Error", err.Error())
				return
			}
			if modelMeta.Task == "" {
				uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
			} else {
				if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
					uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(val)
				} else {
					util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
					makeJSONResponse(w, 400, "Add Model Error", "README.md contains unsupported task")
					return
				}
			}
		} else {
			uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		}

		maxBatchSize, err := util.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"Local model",
				"Missing ensemble model",
				"",
				"err.Error()",
			)
			if e != nil {
				logger.Error(e.Error())
			}
			obj, _ := json.Marshal(st.Details())
			makeJSONResponse(w, 400, st.Message(), string(obj))
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			return
		}

		allowedMaxBatchSize := 0
		switch uploadedModel.Instances[0].Task {
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_CLASSIFICATION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_DETECTION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_KEYPOINT):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_OCR):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
		}

		if maxBatchSize > allowedMaxBatchSize {
			st, e := sterr.CreateErrorPreconditionFailure(
				"[handler] create a model",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "MAX BATCH SIZE LIMITATION",
						Subject:     "Create a model error",
						Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
					},
				})
			if e != nil {
				logger.Error(e.Error())
			}
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			obj, _ := json.Marshal(st.Details())
			makeJSONResponse(w, 400, st.Message(), string(obj))
			return
		}

		dbModel, err := modelService.CreateModel(owner, &uploadedModel)
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJSONResponse(w, 500, "Add Model Error", err.Error())
			return
		}

		pbModel := DBModelToPBModel(&localModelDefinition, dbModel)

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(201)

		m := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: false, EmitUnpopulated: true}
		b, err := m.Marshal(&modelPB.CreateModelResponse{Model: pbModel})
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJSONResponse(w, 500, "Add Model Error", err.Error())
			return
		}
		_, _ = w.Write(b)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

// AddModel - upload a model to the model server
func (h *handler) CreateModelBinaryFileUpload(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (err error) {
	logger, _ := logger.GetZapLogger()

	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}
	tmpFile, uploadedModel, modelDefID, err := saveFile(stream)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	_, err = h.service.GetModelById(owner, uploadedModel.ID, modelPB.View_VIEW_FULL)
	if err == nil {
		return status.Errorf(codes.AlreadyExists, fmt.Sprintf("The model %v already existed", uploadedModel.ID))
	}

	modelDef, err := h.service.GetModelDefinition(modelDefID)
	if err != nil {
		return err
	}
	uploadedModel.ModelDefinitionUid = modelDef.UID

	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, DBModelToPBModel(&modelDef, uploadedModel), true); err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}

	uploadedModel.Owner = owner

	// extract zip file from tmp to models directory
	readmeFilePath, ensembleFilePath, err := unzip(tmpFile, config.Config.TritonServer.ModelStore, owner, uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return status.Errorf(codes.Internal, err.Error())
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		if modelMeta.Task == "" {
			uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		} else {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(val)
			} else {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
				return status.Errorf(codes.InvalidArgument, "README.md contains unsupported task")
			}
		}
	} else {
		uploadedModel.Instances[0].Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
	}

	maxBatchSize, err := util.GetMaxBatchSize(ensembleFilePath)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"Local model",
			"Missing ensemble model",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return st.Err()
	}

	allowedMaxBatchSize := 0
	switch uploadedModel.Instances[0].Task {
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_CLASSIFICATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_DETECTION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_KEYPOINT):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_OCR):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
	}

	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})
		if e != nil {
			logger.Error(e.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return st.Err()
	}

	dbModel, err := h.service.CreateModel(owner, uploadedModel)
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return err
	}
	pbModel := DBModelToPBModel(&modelDef, dbModel)
	err = stream.SendAndClose(&modelPB.CreateModelBinaryFileUploadResponse{Model: pbModel})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return
}

func createGitHubModel(h *handler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
	logger, _ := logger.GetZapLogger()

	var modelConfig datamodel.GitHubModelConfiguration
	b, err := req.Model.Configuration.MarshalJSON()
	if err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.Repository == "" {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub URL")
	}
	githubInfo, err := util.GetGitHubRepoInfo(modelConfig.Repository)
	if err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub Info")
	}
	if len(githubInfo.Tags) == 0 {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "There is no tag in GitHub repository")
	}
	visibility := util.Visibility[githubInfo.Visibility]
	if req.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	} else if req.Model.Visibility == modelPB.Model_VISIBILITY_PRIVATE {
		visibility = modelPB.Model_VISIBILITY_PRIVATE
	}
	bModelConfig, _ := json.Marshal(datamodel.GitHubModelConfiguration{
		Repository: modelConfig.Repository,
		HtmlUrl:    "https://github.com/" + modelConfig.Repository,
	})
	githubModel := datamodel.Model{
		ID:                 req.Model.Id,
		ModelDefinitionUid: modelDefinition.UID,
		Owner:              owner,
		Visibility:         datamodel.ModelVisibility(visibility),
		Description:        githubInfo.Description,
		Configuration:      bModelConfig,
		Instances:          []datamodel.ModelInstance{},
	}
	for _, tag := range githubInfo.Tags {
		instanceConfig := datamodel.GitHubModelInstanceConfiguration{
			Repository: modelConfig.Repository,
			HtmlUrl:    "https://github.com/" + modelConfig.Repository + "/tree/" + tag.Name,
			Tag:        tag.Name,
		}
		rdid, _ := uuid.NewV4()
		modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
		err = util.GitHubCloneWOLargeFile(modelSrcDir, instanceConfig)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"GitHub",
				"Clone repository",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, tag.Name)
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		bInstanceConfig, _ := json.Marshal(instanceConfig)
		instance := datamodel.ModelInstance{
			ID:            tag.Name,
			State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			Configuration: bInstanceConfig,
		}

		readmeFilePath, ensembleFilePath, err := updateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, githubModel.ID, &instance)
		_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"Model folder structure",
				"",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, tag.Name)
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		if _, err := os.Stat(readmeFilePath); err == nil {
			modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
			if err != nil || modelMeta.Task == "" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					"[handler] create a model error",
					"REAME.md file",
					"Could not get meta data from README.md file",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, tag.Name)
				return &modelPB.CreateModelResponse{}, st.Err()
			}
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				instance.Task = datamodel.ModelInstanceTask(val)
			} else {
				if modelMeta.Task != "" {
					st, err := sterr.CreateErrorResourceInfo(
						codes.FailedPrecondition,
						"[handler] create a model error",
						"REAME.md file",
						"README.md contains unsupported task",
						"",
						err.Error(),
					)
					if err != nil {
						logger.Error(err.Error())
					}
					util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, instance.ID)
					return &modelPB.CreateModelResponse{}, st.Err()
				} else {
					instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
				}
			}
		} else {
			instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		}

		maxBatchSize, err := util.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"GitHub model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			return &modelPB.CreateModelResponse{}, st.Err()
		}

		allowedMaxBatchSize := 0
		switch instance.Task {
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_CLASSIFICATION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_DETECTION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_KEYPOINT):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_OCR):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
		}
		if maxBatchSize > allowedMaxBatchSize {
			st, e := sterr.CreateErrorPreconditionFailure(
				"[handler] create a model",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "MAX BATCH SIZE LIMITATION",
						Subject:     "Create a model error",
						Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
					},
				})
			if e != nil {
				logger.Error(e.Error())
			}
			return &modelPB.CreateModelResponse{}, st.Err()
		}

		githubModel.Instances = append(githubModel.Instances, instance)
	}
	dbModel, err := h.service.CreateModel(owner, &githubModel)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[handler] create a model error",
			"Model service",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		for _, tag := range githubInfo.Tags {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, tag.Name)
		}
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	pbModel := DBModelToPBModel(modelDefinition, dbModel)
	return &modelPB.CreateModelResponse{Model: pbModel}, nil
}

func createArtiVCModel(h *handler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
	logger, _ := logger.GetZapLogger()

	var modelConfig datamodel.ArtiVCModelConfiguration
	b, err := req.Model.GetConfiguration().MarshalJSON()
	if err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.Url == "" {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub URL")
	}

	visibility := modelPB.Model_VISIBILITY_PRIVATE
	if req.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	}
	bModelConfig, _ := json.Marshal(modelConfig)
	description := ""
	if req.Model.Description != nil {
		description = *req.Model.Description
	}
	artivcModel := datamodel.Model{
		ID:                 req.Model.Id,
		ModelDefinitionUid: modelDefinition.UID,
		Owner:              owner,
		Visibility:         datamodel.ModelVisibility(visibility),
		Description:        description,
		Configuration:      bModelConfig,
		Instances:          []datamodel.ModelInstance{},
	}
	rdid, _ := uuid.NewV4()
	tmpDir := fmt.Sprintf("./%s", rdid.String())
	tags, err := util.ArtiVCGetTags(tmpDir, modelConfig)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"ArtiVC",
			"Get tags",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	_ = os.RemoveAll(tmpDir)
	for _, tag := range tags {
		instanceConfig := datamodel.ArtiVCModelInstanceConfiguration{
			Url: modelConfig.Url,
			Tag: tag,
		}
		rdid, _ := uuid.NewV4()
		modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
		err = util.ArtiVCClone(modelSrcDir, modelConfig, instanceConfig, false)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"ArtiVC",
				"Clone repository",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			_ = os.RemoveAll(modelSrcDir)
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		util.AddMissingTritonModelFolder(modelSrcDir) // large files not pull then need to create triton model folder
		bInstanceConfig, _ := json.Marshal(instanceConfig)
		instance := datamodel.ModelInstance{
			ID:            tag,
			State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			Configuration: bInstanceConfig,
		}

		readmeFilePath, ensembleFilePath, err := updateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, artivcModel.ID, &instance)
		_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"Model folder structure",
				"",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		if _, err := os.Stat(readmeFilePath); err == nil {
			modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
			if err != nil || modelMeta.Task == "" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					"[handler] create a model error",
					"REAME.md file",
					"Could not get meta data from README.md file",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
				return &modelPB.CreateModelResponse{}, st.Err()
			}
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				instance.Task = datamodel.ModelInstanceTask(val)
			} else {
				if modelMeta.Task != "" {
					st, err := sterr.CreateErrorResourceInfo(
						codes.FailedPrecondition,
						"[handler] create a model error",
						"REAME.md file",
						"README.md contains unsupported task",
						"",
						err.Error(),
					)
					if err != nil {
						logger.Error(err.Error())
					}
					util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, instance.ID)
					return &modelPB.CreateModelResponse{}, st.Err()
				} else {
					instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
				}
			}
		} else {
			instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
		}

		maxBatchSize, err := util.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"ArtiVC model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			return &modelPB.CreateModelResponse{}, st.Err()
		}

		allowedMaxBatchSize := 0
		switch instance.Task {
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_CLASSIFICATION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_DETECTION):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_KEYPOINT):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
		case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_OCR):
			allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
		}
		if maxBatchSize > allowedMaxBatchSize {
			st, e := sterr.CreateErrorPreconditionFailure(
				"[handler] create a model",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "MAX BATCH SIZE LIMITATION",
						Subject:     "Create a model error",
						Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
					},
				})

			if e != nil {
				logger.Error(e.Error())
			}
			return &modelPB.CreateModelResponse{}, st.Err()
		}

		artivcModel.Instances = append(artivcModel.Instances, instance)
	}
	dbModel, err := h.service.CreateModel(owner, &artivcModel)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[handler] create a model error",
			"Model service",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		for _, tag := range tags {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
		}
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	pbModel := DBModelToPBModel(modelDefinition, dbModel)

	return &modelPB.CreateModelResponse{Model: pbModel}, nil
}

func createHuggingFaceModel(h *handler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
	logger, _ := logger.GetZapLogger()

	var modelConfig datamodel.HuggingFaceModelConfiguration
	b, err := req.Model.GetConfiguration().MarshalJSON()
	if err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.RepoId == "" {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid model ID")
	}
	modelConfig.HtmlUrl = "https://huggingface.co/" + modelConfig.RepoId

	visibility := modelPB.Model_VISIBILITY_PRIVATE
	if req.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	}
	bModelConfig, _ := json.Marshal(modelConfig)
	description := ""
	if req.Model.Description != nil {
		description = *req.Model.Description
	}
	huggingfaceModel := datamodel.Model{
		ID:                 req.Model.Id,
		ModelDefinitionUid: modelDefinition.UID,
		Owner:              owner,
		Visibility:         datamodel.ModelVisibility(visibility),
		Description:        description,
		Configuration:      bModelConfig,
		Instances:          []datamodel.ModelInstance{},
	}
	rdid, _ := uuid.NewV4()
	configTmpDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if err := util.HuggingFaceClone(configTmpDir, modelConfig); err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"GitHub",
			"Clone model repository",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		_ = os.RemoveAll(configTmpDir)
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	rdid, _ = uuid.NewV4()
	modelDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if err := util.GenerateHuggingFaceModel(configTmpDir, modelDir, req.Model.Id); err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"GitHub",
			"Generate HuggingFace model",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		_ = os.RemoveAll(modelDir)
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	_ = os.RemoveAll(configTmpDir)
	instanceConfig := datamodel.HuggingFaceModelInstanceConfiguration{
		RepoId:  modelConfig.RepoId,
		HtmlUrl: modelConfig.HtmlUrl + "/tree/main",
	}
	bInstanceConfig, _ := json.Marshal(instanceConfig)

	instance := datamodel.ModelInstance{
		ID:            "latest",
		State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
		Configuration: bInstanceConfig,
	}

	readmeFilePath, ensembleFilePath, err := updateModelPath(modelDir, config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, &instance)
	_ = os.RemoveAll(modelDir) // remove uploaded temporary files
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"Model folder structure",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, "latest")
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"REAME.md file",
				"Could not get meta data from README.md file",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, "latest")
			return &modelPB.CreateModelResponse{}, st.Err()
		}

		if modelMeta.Task != "" {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				instance.Task = datamodel.ModelInstanceTask(val)
			} else {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					"[handler] create a model error",
					"REAME.md file",
					"README.md contains unsupported task",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, instance.ID)
				return &modelPB.CreateModelResponse{}, st.Err()
			}
		} else {
			if len(modelMeta.Tags) == 0 {
				instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
			} else { // check in tags also for HuggingFace model card README.md
				instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
				for _, tag := range modelMeta.Tags {
					if val, ok := util.Tags[strings.ToUpper(tag)]; ok {
						instance.Task = datamodel.ModelInstanceTask(val)
						break
					}
				}
			}
		}

	} else {
		instance.Task = datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED)
	}

	maxBatchSize, err := util.GetMaxBatchSize(ensembleFilePath)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] create a model error",
			"HuggingFace model",
			"Missing ensemble model",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	allowedMaxBatchSize := 0
	switch instance.Task {
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_UNSPECIFIED):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_CLASSIFICATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_DETECTION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_KEYPOINT):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
	case datamodel.ModelInstanceTask(modelPB.ModelInstance_TASK_OCR):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
	}
	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})
		if e != nil {
			logger.Error(e.Error())
		}
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	huggingfaceModel.Instances = append(huggingfaceModel.Instances, instance)

	dbModel, err := h.service.CreateModel(owner, &huggingfaceModel)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[handler] create a model error",
			"Model service",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, "latest")
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	pbModel := DBModelToPBModel(modelDefinition, dbModel)

	return &modelPB.CreateModelResponse{Model: pbModel}, nil
}

func (h *handler) CreateModel(ctx context.Context, req *modelPB.CreateModelRequest) (*modelPB.CreateModelResponse, error) {
	resp := &modelPB.CreateModelResponse{}
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}
	// Set all OUTPUT_ONLY fields to zero value on the requested payload model resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Model, outputOnlyFields); err != nil {
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Return error if REQUIRED fields are not provided in the requested payload model resource
	if err := checkfield.CheckRequiredFields(req.Model, requiredFields); err != nil {
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.Model.GetId()); err != nil {
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, req.Model, false); err != nil {
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}
	_, err = h.service.GetModelById(owner, req.Model.Id, modelPB.View_VIEW_FULL)
	if err == nil {
		return resp, status.Errorf(codes.AlreadyExists, "Model already existed")
	}

	if req.Model.Configuration == nil {
		return resp, status.Errorf(codes.InvalidArgument, "Missing Configuration")
	}

	modelDefinitionID, err := resource.GetDefinitionID(req.Model.ModelDefinition)
	if err != nil {
		return resp, err
	}

	modelDefinition, err := h.service.GetModelDefinition(modelDefinitionID)
	if err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// validate model configuration
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(modelDefinition.ModelSpec.String()), rs); err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Could not get model definition")
	}
	if err := datamodel.ValidateJSONSchema(rs, req.Model.GetConfiguration(), true); err != nil {
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
	}

	switch modelDefinitionID {
	case "github":
		return createGitHubModel(h, ctx, req, owner, &modelDefinition)
	case "artivc":
		return createArtiVCModel(h, ctx, req, owner, &modelDefinition)
	case "huggingface":
		return createHuggingFaceModel(h, ctx, req, owner, &modelDefinition)
	default:
		return resp, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
	}
}

func (h *handler) ListModel(ctx context.Context, req *modelPB.ListModelRequest) (*modelPB.ListModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.ListModelResponse{}, err
	}
	dbModels, nextPageToken, totalSize, err := h.service.ListModel(owner, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelResponse{}, err
	}

	pbModels := []*modelPB.Model{}
	for _, dbModel := range dbModels {
		modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.ListModelResponse{}, err
		}
		pbModels = append(pbModels, DBModelToPBModel(&modelDef, &dbModel))
	}

	resp := modelPB.ListModelResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *handler) LookUpModel(ctx context.Context, req *modelPB.LookUpModelRequest) (*modelPB.LookUpModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.LookUpModelResponse{}, err
	}
	sUID, err := resource.GetID(req.Permalink)
	if err != nil {
		return &modelPB.LookUpModelResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		return &modelPB.LookUpModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbModel, err := h.service.GetModelByUid(owner, uid, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.LookUpModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.LookUpModelResponse{Model: pbModel}, nil
}

func (h *handler) GetModel(ctx context.Context, req *modelPB.GetModelRequest) (*modelPB.GetModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	dbModel, err := h.service.GetModelById(owner, id, req.GetView())
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.GetModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.GetModelResponse{Model: pbModel}, err
}

func (h *handler) UpdateModel(ctx context.Context, req *modelPB.UpdateModelRequest) (*modelPB.UpdateModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.UpdateModelResponse{}, err
	}
	id, err := resource.GetID(req.Model.Name)
	if err != nil {
		return &modelPB.UpdateModelResponse{}, err
	}
	dbModel, err := h.service.GetModelById(owner, id, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UpdateModelResponse{}, err
	}
	updateModel := datamodel.Model{
		ID: id,
	}
	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		for _, field := range req.UpdateMask.Paths {
			switch field {
			case "description":
				updateModel.Description = *req.Model.Description
			}
		}
	}
	dbModel, err = h.service.UpdateModel(dbModel.UID, &updateModel)
	if err != nil {
		return &modelPB.UpdateModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.UpdateModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.UpdateModelResponse{Model: pbModel}, err
}

func (h *handler) DeleteModel(ctx context.Context, req *modelPB.DeleteModelRequest) (*modelPB.DeleteModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.DeleteModelResponse{}, err
	}
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.DeleteModelResponse{}, err
	}

	// Manually set the custom header to have a StatusNoContent http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &modelPB.DeleteModelResponse{}, h.service.DeleteModel(owner, id)
}

func (h *handler) RenameModel(ctx context.Context, req *modelPB.RenameModelRequest) (*modelPB.RenameModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	dbModel, err := h.service.RenameModel(owner, id, req.NewModelId)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.RenameModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.RenameModelResponse{Model: pbModel}, nil
}

func (h *handler) PublishModel(ctx context.Context, req *modelPB.PublishModelRequest) (*modelPB.PublishModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.PublishModelResponse{}, err
	}
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.PublishModelResponse{}, err
	}
	dbModel, err := h.service.PublishModel(owner, id)
	if err != nil {
		return &modelPB.PublishModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.PublishModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.PublishModelResponse{Model: pbModel}, nil
}

func (h *handler) UnpublishModel(ctx context.Context, req *modelPB.UnpublishModelRequest) (*modelPB.UnpublishModelResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.UnpublishModelResponse{}, err
	}
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.UnpublishModelResponse{}, err
	}
	dbModel, err := h.service.UnpublishModel(owner, id)
	if err != nil {
		return &modelPB.UnpublishModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.UnpublishModelResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel)
	return &modelPB.UnpublishModelResponse{Model: pbModel}, nil
}

func (h *handler) GetModelInstance(ctx context.Context, req *modelPB.GetModelInstanceRequest) (*modelPB.GetModelInstanceResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	modelID, instanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	dbModel, err := h.service.GetModelById(owner, modelID, req.GetView())
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, req.GetView())
	if err != nil {
		return &modelPB.GetModelInstanceResponse{}, err
	}

	pbModelInstance := DBModelInstanceToPBModelInstance(&modelDef, &dbModel, &dbModelInstance)
	return &modelPB.GetModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (h *handler) LookUpModelInstance(ctx context.Context, req *modelPB.LookUpModelInstanceRequest) (*modelPB.LookUpModelInstanceResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, err
	}
	sModelUID, sInstanceUID, err := resource.GetModelInstanceID(req.Permalink)
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	modelUID, err := uuid.FromString(sModelUID)
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbModel, err := h.service.GetModelByUid(owner, modelUID, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, status.Error(codes.NotFound, err.Error())
	}
	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, err
	}
	instanceUID, err := uuid.FromString(sInstanceUID)
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbModelInstance, err := h.service.GetModelInstanceByUid(dbModel.UID, instanceUID, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelInstanceResponse{}, status.Error(codes.NotFound, err.Error())
	}

	pbModelInstance := DBModelInstanceToPBModelInstance(&modelDef, &dbModel, &dbModelInstance)
	return &modelPB.LookUpModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (h *handler) ListModelInstance(ctx context.Context, req *modelPB.ListModelInstanceRequest) (*modelPB.ListModelInstanceResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	modelID, err := resource.GetID(req.Parent)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}
	modelInDB, err := h.service.GetModelById(owner, modelID, req.GetView())
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(modelInDB.ModelDefinitionUid)
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	dbModelInstances, nextPageToken, totalSize, err := h.service.ListModelInstance(modelInDB.UID, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelInstanceResponse{}, err
	}

	pbInstances := []*modelPB.ModelInstance{}
	for _, dbModelInstance := range dbModelInstances {
		pbInstances = append(pbInstances, DBModelInstanceToPBModelInstance(&modelDef, &modelInDB, &dbModelInstance))
	}

	resp := modelPB.ListModelInstanceResponse{
		Instances:     pbInstances,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *handler) DeployModelInstance(ctx context.Context, req *modelPB.DeployModelInstanceRequest) (*modelPB.DeployModelInstanceResponse, error) {
	logger, _ := logger.GetZapLogger()

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	modelID, instanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	dbModel, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}
	tritonModels, err := h.service.GetTritonModels(dbModelInstance.UID)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	// downloading model weight when making inference
	switch modelDef.ID {
	case "github":
		if !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var instanceConfig datamodel.GitHubModelInstanceConfiguration
			if err := json.Unmarshal(dbModelInstance.Configuration, &instanceConfig); err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.Internal, err.Error())
			}
			rdid, _ := uuid.NewV4()
			modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
			if err := util.GitHubCloneWLargeFile(modelSrcDir, instanceConfig); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.Internal, err.Error())
			}
			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.Internal, err.Error())
			}
			_ = os.RemoveAll(modelSrcDir)
		}
	case "huggingface":
		if !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var instanceConfig datamodel.HuggingFaceModelInstanceConfiguration
			if err := json.Unmarshal(dbModelInstance.Configuration, &instanceConfig); err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.Internal, err.Error())
			}

			var modelConfig datamodel.HuggingFaceModelConfiguration
			err = json.Unmarshal([]byte(dbModel.Configuration), &modelConfig)
			if err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Errorf(codes.Internal, err.Error())
			}

			rdid, _ := uuid.NewV4()
			modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
			if err = util.HuggingFaceExport(modelSrcDir, modelConfig, dbModel.ID); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Errorf(codes.Internal, fmt.Sprintf("Export model error %v", err.Error()))
			}
			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
			}

			if err := util.UpdateModelConfig(config.Config.TritonServer.ModelStore, tritonModels); err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
			}
			_ = os.RemoveAll(modelSrcDir)
		}
	case "artivc":
		if !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var instanceConfig datamodel.ArtiVCModelInstanceConfiguration
			if err := json.Unmarshal(dbModelInstance.Configuration, &instanceConfig); err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.Internal, err.Error())
			}

			var modelConfig datamodel.ArtiVCModelConfiguration
			err = json.Unmarshal([]byte(dbModel.Configuration), &modelConfig)
			if err != nil {
				return &modelPB.DeployModelInstanceResponse{}, status.Errorf(codes.Internal, err.Error())
			}

			rdid, _ := uuid.NewV4()
			modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
			err = util.ArtiVCClone(modelSrcDir, modelConfig, instanceConfig, true)
			if err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
			}
			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return &modelPB.DeployModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
			}
		}
	}
	err = h.service.DeployModelInstance(dbModelInstance.UID)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[handler] deploy model error",
			"triton-inference-server",
			"deploy model",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] deploy model error",
				"triton-inference-server",
				"Out of memory for deploying the model to triton server, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}

		return &modelPB.DeployModelInstanceResponse{}, st.Err()
	}

	dbModelInstance, err = h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}
	pbModelInstance := DBModelInstanceToPBModelInstance(&modelDef, &dbModel, &dbModelInstance)

	return &modelPB.DeployModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (h *handler) UndeployModelInstance(ctx context.Context, req *modelPB.UndeployModelInstanceRequest) (*modelPB.UndeployModelInstanceResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	modelID, instanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModel, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	err = h.service.UndeployModelInstance(dbModelInstance.UID)
	if err != nil {
		// Manually set the custom header to have a StatusUnprocessableEntity http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusUnprocessableEntity))); err != nil {
			return &modelPB.UndeployModelInstanceResponse{}, status.Errorf(codes.Internal, err.Error())
		}
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	dbModelInstance, err = h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}
	pbModelInstance := DBModelInstanceToPBModelInstance(&modelDef, &dbModel, &dbModelInstance)

	return &modelPB.UndeployModelInstanceResponse{Instance: pbModelInstance}, nil
}

func (h *handler) TestModelInstanceBinaryFileUpload(stream modelPB.ModelService_TestModelInstanceBinaryFileUploadServer) error {
	logger, _ := logger.GetZapLogger()

	if !h.triton.IsTritonServerReady() {
		return status.Error(codes.Unavailable, "Triton Server not ready yet")
	}

	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}

	imageBytes, modelID, instanceID, err := savePredictInputsTestMode(stream)
	if err != nil {
		return status.Error(codes.Internal, "Could not save the file")
	}

	modelInDB, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}
	modelInstanceInDB, err := h.service.GetModelInstance(modelInDB.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	// check whether model support batching or not. If not, raise an error
	if len(imageBytes) > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(modelInstanceInDB.UID)
		if err != nil {
			return err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			return status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := h.service.ModelInferTestMode(owner, modelInstanceInDB.UID, imageBytes, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] inference model error",
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		return st.Err()
	}

	err = stream.SendAndClose(&modelPB.TestModelInstanceBinaryFileUploadResponse{
		Task:         task,
		BatchOutputs: response,
	})
	return err
}

func (h *handler) TriggerModelInstanceBinaryFileUpload(stream modelPB.ModelService_TriggerModelInstanceBinaryFileUploadServer) error {
	logger, _ := logger.GetZapLogger()

	if !h.triton.IsTritonServerReady() {
		return status.Error(codes.Unavailable, "Triton Server not ready yet")
	}

	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}

	imgsBytes, modelID, instanceID, err := savePredictInputsTriggerMode(stream)
	if err != nil {
		return status.Error(codes.Internal, "Could not save the file")
	}

	modelInDB, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}
	modelInstanceInDB, err := h.service.GetModelInstance(modelInDB.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	// check whether model support batching or not. If not, raise an error
	if len(imgsBytes) > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(modelInstanceInDB.UID)
		if err != nil {
			return err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			return status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := h.service.ModelInfer(modelInstanceInDB.UID, imgsBytes, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] inference model error",
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		return st.Err()
	}

	err = stream.SendAndClose(&modelPB.TriggerModelInstanceBinaryFileUploadResponse{
		Task:         task,
		BatchOutputs: response,
	})
	return err
}

func (h *handler) TriggerModelInstance(ctx context.Context, req *modelPB.TriggerModelInstanceRequest) (*modelPB.TriggerModelInstanceResponse, error) {
	logger, _ := logger.GetZapLogger()

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	modelID, modelInstanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	modelInDB, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	modelInstanceInDB, err := h.service.GetModelInstance(modelInDB.UID, modelInstanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, err
	}

	imgsBytes, _, err := parseImageRequestInputsToBytes(req)
	if err != nil {
		return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// check whether model support batching or not. If not, raise an error
	if len(imgsBytes) > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(modelInstanceInDB.UID)
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := h.service.ModelInfer(modelInstanceInDB.UID, imgsBytes, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] inference model error",
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		return &modelPB.TriggerModelInstanceResponse{}, st.Err()
	}

	return &modelPB.TriggerModelInstanceResponse{
		Task:         task,
		BatchOutputs: response,
	}, nil
}

func (h *handler) TestModelInstance(ctx context.Context, req *modelPB.TestModelInstanceRequest) (*modelPB.TestModelInstanceResponse, error) {
	logger, _ := logger.GetZapLogger()

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.TestModelInstanceResponse{}, err
	}

	modelID, modelInstanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.TestModelInstanceResponse{}, err
	}

	modelInDB, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.TestModelInstanceResponse{}, err
	}

	modelInstanceInDB, err := h.service.GetModelInstance(modelInDB.UID, modelInstanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.TestModelInstanceResponse{}, err
	}

	imgsBytes, _, err := parseImageRequestInputsToBytes(&modelPB.TriggerModelInstanceRequest{
		Name:   req.Name,
		Inputs: req.Inputs,
	})
	if err != nil {
		return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// check whether model support batching or not. If not, raise an error
	if len(imgsBytes) > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(modelInstanceInDB.UID)
		if err != nil {
			return &modelPB.TestModelInstanceResponse{}, err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
	response, err := h.service.ModelInferTestMode(owner, modelInstanceInDB.UID, imgsBytes, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			"[handler] inference model error",
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		return &modelPB.TestModelInstanceResponse{}, st.Err()
	}

	return &modelPB.TestModelInstanceResponse{
		Task:         task,
		BatchOutputs: response,
	}, nil
}

func inferModelInstanceByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string, mode string) {
	logger, _ := logger.GetZapLogger()

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		owner, err := resource.GetOwnerFromHeader(r)
		if err != nil || owner == "" {
			makeJSONResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
			return
		}

		instanceName := pathParams["name"]
		if instanceName == "" {
			makeJSONResponse(w, 422, "Required parameter missing", "Required parameter mode name not found")
			return
		}

		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		defer tritonService.Close()
		pipelineServiceClient, pipelineServiceClientConn := external.InitPipelineServiceClient()
		defer pipelineServiceClientConn.Close()
		redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
		defer redisClient.Close()
		modelService := service.NewService(modelRepository, tritonService, pipelineServiceClient, redisClient)

		modelID, instanceID, err := resource.GetModelInstanceID(instanceName)
		if err != nil {
			makeJSONResponse(w, 400, "Parameter invalid", "Required parameter instance_name is invalid")
			return
		}

		modelInDB, err := modelService.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
		if err != nil {
			makeJSONResponse(w, 404, "Model not found", "The model not found in server")
			return
		}

		modelInstanceInDB, err := modelService.GetModelInstance(modelInDB.UID, instanceID, modelPB.View_VIEW_FULL)
		if err != nil {
			makeJSONResponse(w, 404, "Model instance not found", "The model instance not found in server")
			return
		}

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJSONResponse(w, 400, "Internal Error", fmt.Sprintf("Error while reading file from request %v", err))
			return
		}

		imgsBytes, _, err := parseImageFormDataInputsToBytes(r)
		if err != nil {
			makeJSONResponse(w, 400, "File Input Error", err.Error())
			return
		}

		// check whether model support batching or not. If not, raise an error
		if len(imgsBytes) > 1 {
			tritonModelInDB, err := modelService.GetTritonEnsembleModel(modelInstanceInDB.UID)
			if err != nil {
				makeJSONResponse(w, 404, "Triton Model Error", fmt.Sprintf("The triton model corresponding to instance %v do not exist", modelInstanceInDB.ID))
				return
			}
			configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
			doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
			if err != nil {
				makeJSONResponse(w, 400, "Batching Support Error", err.Error())
				return
			}
			if !doSupportBatch {
				makeJSONResponse(w, 400, "Batching Support Error", "The model do not support batching, so could not make inference with multiple images")
				return
			}
		}
		task := modelPB.ModelInstance_Task(modelInstanceInDB.Task)
		var response []*modelPB.BatchOutput
		if mode == "test" {
			response, err = modelService.ModelInferTestMode(owner, modelInstanceInDB.UID, imgsBytes, task)
		} else {
			response, err = modelService.ModelInfer(modelInstanceInDB.UID, imgsBytes, task)
		}
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] inference model error",
				"Triton inference server",
				"",
				"",
				err.Error(),
			)
			if strings.Contains(err.Error(), "Failed to allocate memory") {
				st, e = sterr.CreateErrorResourceInfo(
					codes.ResourceExhausted,
					"[handler] inference model error",
					"Triton inference server OOM",
					"Out of memory for running the model, maybe try with smaller batch size",
					"",
					err.Error(),
				)
			}

			if e != nil {
				logger.Error(e.Error())
			}
			obj, _ := json.Marshal(st.Details())
			makeJSONResponse(w, 500, st.Message(), string(obj))
			return
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		res, err := util.MarshalOptions.Marshal(&modelPB.TestModelInstanceBinaryFileUploadResponse{
			Task:         task,
			BatchOutputs: response,
		})
		if err != nil {
			makeJSONResponse(w, 500, "Error Predict Model", err.Error())
			return
		}
		_, _ = w.Write(res)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

func HandleTestModelInstanceByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelInstanceByUpload(w, r, pathParams, "test")
}

func HandleTriggerModelInstanceByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelInstanceByUpload(w, r, pathParams, "trigger")
}

func (h *handler) GetModelInstanceCard(ctx context.Context, req *modelPB.GetModelInstanceCardRequest) (*modelPB.GetModelInstanceCardResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	modelID, instanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	dbModel, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	_, err = h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.GetModelInstanceCardResponse{}, err
	}

	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md#%v", config.Config.TritonServer.ModelStore, owner, modelID, instanceID)
	stat, err := os.Stat(readmeFilePath)
	if err != nil { // return empty content base64
		return &modelPB.GetModelInstanceCardResponse{
			Readme: &modelPB.ModelInstanceCard{
				Name:     req.Name,
				Size:     0,
				Type:     "file",
				Encoding: "base64",
				Content:  []byte(""),
			},
		}, nil
	}

	f, _ := os.Open(readmeFilePath)
	reader := bufio.NewReader(f)
	content, _ := ioutil.ReadAll(reader)

	return &modelPB.GetModelInstanceCardResponse{Readme: &modelPB.ModelInstanceCard{
		Name:     req.Name,
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Content:  []byte(content),
	}}, nil
}

func (h *handler) GetModelDefinition(ctx context.Context, req *modelPB.GetModelDefinitionRequest) (*modelPB.GetModelDefinitionResponse, error) {
	definitionID, err := resource.GetDefinitionID(req.Name)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	dbModelDefinition, err := h.service.GetModelDefinition(definitionID)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	pbModelInstance := DBModelDefinitionToPBModelDefinition(&dbModelDefinition)
	return &modelPB.GetModelDefinitionResponse{ModelDefinition: pbModelInstance}, nil
}

func (h *handler) ListModelDefinition(ctx context.Context, req *modelPB.ListModelDefinitionRequest) (*modelPB.ListModelDefinitionResponse, error) {

	dbModelDefinitions, nextPageToken, totalSize, err := h.service.ListModelDefinition(req.GetView(), int(req.GetPageSize()), req.GetPageToken())
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
