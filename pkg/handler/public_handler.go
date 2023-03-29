package handler

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.temporal.io/sdk/client"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/util"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"
	"github.com/instill-ai/x/zapadapter"

	"google.golang.org/protobuf/types/known/anypb"

	database "github.com/instill-ai/model-backend/pkg/db"
	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"Id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"Name", "Uid", "Visibility", "Owner", "CreateTime", "UpdateTime"}

type PublicHandler struct {
	modelPB.UnimplementedModelPublicServiceServer
	service service.Service
	triton  triton.Triton
}

func NewPublicHandler(s service.Service, t triton.Triton) modelPB.ModelPublicServiceServer {
	datamodel.InitJSONSchema()
	return &PublicHandler{
		service: s,
		triton:  t,
	}
}

func savePredictInputsTriggerMode(stream modelPB.ModelPublicService_TriggerModelInstanceBinaryFileUploadServer) (triggerInput interface{}, modelID string, instanceID string, err error) {

	var firstChunk = true

	var fileData *modelPB.TriggerModelInstanceBinaryFileUploadRequest

	var allContentFiles []byte
	var fileLengths []uint64

	var textToImageInput *triton.TextToImageInput
	var textGeneration *triton.TextGenerationInput

	var task *modelPB.TaskInputStream
	for {
		fileData, err = stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return nil, "", "", err
		}

		if firstChunk { //first chunk contains model instance name
			firstChunk = false
			modelID, instanceID, err = resource.GetModelInstanceID(fileData.Name) // format "models/{model}/instances/{instance}"
			if err != nil {
				return nil, "", "", err
			}
			task = fileData.TaskInput
			switch fileData.TaskInput.Input.(type) {
			case *modelPB.TaskInputStream_Classification:
				fileLengths = fileData.TaskInput.GetClassification().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
			case *modelPB.TaskInputStream_Detection:
				fileLengths = fileData.TaskInput.GetDetection().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
			case *modelPB.TaskInputStream_Keypoint:
				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
			case *modelPB.TaskInputStream_Ocr:
				fileLengths = fileData.TaskInput.GetOcr().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
			case *modelPB.TaskInputStream_InstanceSegmentation:
				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
			case *modelPB.TaskInputStream_SemanticSegmentation:
				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
			case *modelPB.TaskInputStream_TextToImage:
				textToImageInput = &triton.TextToImageInput{
					Prompt:   fileData.TaskInput.GetTextToImage().Prompt,
					Steps:    *fileData.TaskInput.GetTextToImage().Steps,
					CfgScale: *fileData.TaskInput.GetTextToImage().CfgScale,
					Seed:     *fileData.TaskInput.GetTextToImage().Seed,
					Samples:  *fileData.TaskInput.GetTextToImage().Samples,
				}
			case *modelPB.TaskInputStream_TextGeneration:
				textGeneration = &triton.TextGenerationInput{
					Prompt:        fileData.TaskInput.GetTextGeneration().Prompt,
					OutputLen:     *fileData.TaskInput.GetTextGeneration().OutputLen,
					BadWordsList:  *fileData.TaskInput.GetTextGeneration().BadWordsList,
					StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
					TopK:          *fileData.TaskInput.GetTextGeneration().Topk,
					Seed:          *fileData.TaskInput.GetTextGeneration().Seed,
				}
			default:
				return nil, "", "", fmt.Errorf("unsupported task input type")
			}
		} else {
			switch fileData.TaskInput.Input.(type) {
			case *modelPB.TaskInputStream_Classification:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
			case *modelPB.TaskInputStream_Detection:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
			case *modelPB.TaskInputStream_Keypoint:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
			case *modelPB.TaskInputStream_Ocr:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
			case *modelPB.TaskInputStream_InstanceSegmentation:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
			case *modelPB.TaskInputStream_SemanticSegmentation:
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
			default:
				return nil, "", "", fmt.Errorf("unsupported task input type")
			}
		}
	}

	switch task.Input.(type) {
	case *modelPB.TaskInputStream_Classification,
		*modelPB.TaskInputStream_Detection,
		*modelPB.TaskInputStream_Keypoint,
		*modelPB.TaskInputStream_Ocr,
		*modelPB.TaskInputStream_InstanceSegmentation,
		*modelPB.TaskInputStream_SemanticSegmentation:
		if len(fileLengths) == 0 {
			return nil, "", "", fmt.Errorf("wrong parameter length of files")
		}
		imageBytes := make([][]byte, len(fileLengths))
		start := uint64(0)
		for i := 0; i < len(fileLengths); i++ {
			buff := new(bytes.Buffer)
			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
			if err != nil {
				return nil, "", "", err
			}
			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
			if err != nil {
				return nil, "", "", err
			}
			imageBytes[i] = buff.Bytes()
			start += fileLengths[i]
		}
		return imageBytes, modelID, instanceID, nil
	case *modelPB.TaskInputStream_TextToImage:
		return textToImageInput, modelID, instanceID, nil
	case *modelPB.TaskInputStream_TextGeneration:
		return textGeneration, modelID, instanceID, nil
	}
	return nil, "", "", fmt.Errorf("unsupported task input type")
}

func savePredictInputsTestMode(stream modelPB.ModelPublicService_TestModelInstanceBinaryFileUploadServer) (triggerInput interface{}, modelID string, instanceID string, err error) {
	var firstChunk = true
	var fileData *modelPB.TestModelInstanceBinaryFileUploadRequest

	var textToImageInput *triton.TextToImageInput
	var textGeneration *triton.TextGenerationInput

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
			return nil, "", "", err
		}

		if firstChunk { //first chunk contains file name
			firstChunk = false
			modelID, instanceID, err = resource.GetModelInstanceID(fileData.Name) // format "models/{model}/instances/{instance}"
			if err != nil {
				return nil, "", "", err
			}
			switch fileData.TaskInput.Input.(type) {
			case *modelPB.TaskInputStream_Classification:
				fileLengths = fileData.TaskInput.GetClassification().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
			case *modelPB.TaskInputStream_Detection:
				fileLengths = fileData.TaskInput.GetDetection().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
			case *modelPB.TaskInputStream_Keypoint:
				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
			case *modelPB.TaskInputStream_Ocr:
				fileLengths = fileData.TaskInput.GetOcr().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
			case *modelPB.TaskInputStream_InstanceSegmentation:
				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
			case *modelPB.TaskInputStream_SemanticSegmentation:
				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
			case *modelPB.TaskInputStream_TextToImage:
				textToImageInput = &triton.TextToImageInput{
					Prompt:   fileData.TaskInput.GetTextToImage().Prompt,
					Steps:    *fileData.TaskInput.GetTextToImage().Steps,
					CfgScale: *fileData.TaskInput.GetTextToImage().CfgScale,
					Seed:     *fileData.TaskInput.GetTextToImage().Seed,
					Samples:  *fileData.TaskInput.GetTextToImage().Samples,
				}
			case *modelPB.TaskInputStream_TextGeneration:
				textGeneration = &triton.TextGenerationInput{
					Prompt:        fileData.TaskInput.GetTextGeneration().Prompt,
					OutputLen:     *fileData.TaskInput.GetTextGeneration().OutputLen,
					BadWordsList:  *fileData.TaskInput.GetTextGeneration().BadWordsList,
					StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
					TopK:          *fileData.TaskInput.GetTextGeneration().Topk,
					Seed:          *fileData.TaskInput.GetTextGeneration().Seed,
				}
			default:
				return nil, "", "", fmt.Errorf("unsupported task input type")
			}
		}
		switch fileData.TaskInput.Input.(type) {
		case *modelPB.TaskInputStream_Classification:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
		case *modelPB.TaskInputStream_Detection:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
		case *modelPB.TaskInputStream_Keypoint:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
		case *modelPB.TaskInputStream_Ocr:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
		case *modelPB.TaskInputStream_InstanceSegmentation:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
		case *modelPB.TaskInputStream_SemanticSegmentation:
			allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
		default:
			return nil, "", "", fmt.Errorf("unsupported task input type")
		}
	}

	switch fileData.TaskInput.Input.(type) {
	case *modelPB.TaskInputStream_Classification,
		*modelPB.TaskInputStream_Detection,
		*modelPB.TaskInputStream_Keypoint,
		*modelPB.TaskInputStream_Ocr,
		*modelPB.TaskInputStream_InstanceSegmentation,
		*modelPB.TaskInputStream_SemanticSegmentation:
		if len(fileLengths) == 0 {
			return nil, "", "", fmt.Errorf("wrong parameter length of files")
		}
		imageBytes := make([][]byte, len(fileLengths))
		start := uint64(0)
		for i := 0; i < len(fileLengths); i++ {
			buff := new(bytes.Buffer)
			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
			if err != nil {
				return nil, "", "", err
			}
			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
			if err != nil {
				return nil, "", "", err
			}
			imageBytes[i] = buff.Bytes()
			start += fileLengths[i]
		}
		return imageBytes, modelID, instanceID, nil
	case *modelPB.TaskInputStream_TextToImage:
		return textToImageInput, modelID, instanceID, nil
	case *modelPB.TaskInputStream_TextGeneration:
		return textGeneration, modelID, instanceID, nil
	}
	return nil, "", "", fmt.Errorf("unsupported task input type")

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

func (h *PublicHandler) Liveness(ctx context.Context, pb *modelPB.LivenessRequest) (*modelPB.LivenessResponse, error) {
	if !h.triton.IsTritonServerReady() {
		return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !h.triton.IsTritonServerReady() {
		return &modelPB.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING}}, nil
	}

	if h.service.SearchAttributeReady() != nil {
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
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
			return
		}
		file, fileHeader, err := r.FormFile("content")
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
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
		err = util.WriteToFp(fp, buf.Bytes())
		if err != nil {
			makeJSONResponse(w, 400, "File Error", "Error reading input file")
			return
		}
		db := database.GetConnection()
		modelRepository := repository.NewRepository(db)
		tritonService := triton.NewTriton()
		defer tritonService.Close()
		pipelineServiceClient, pipelineServiceClientConn := external.InitPipelinePublicServiceClient()
		defer pipelineServiceClientConn.Close()
		redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
		defer redisClient.Close()

		temporalClient, err := client.Dial(client.Options{
			// ZapAdapter implements log.Logger interface and can be passed
			// to the client constructor using client using client.Options.
			Logger:    zapadapter.NewZapAdapter(logger),
			HostPort:  config.Config.Temporal.ClientOptions.HostPort,
			Namespace: modelWorker.Namespace,
		})
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer temporalClient.Close()

		modelPublicService := service.NewService(modelRepository, tritonService, pipelineServiceClient, redisClient, temporalClient)

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
			Description: sql.NullString{
				String: r.FormValue("description"),
				Valid:  true,
			},
			Configuration: bModelConfig,
		}

		// Validate ModelDefinition JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, DBModelToPBModel(&localModelDefinition, &uploadedModel), true); err != nil {
			makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model configuration is invalid %v", err.Error()))
			return
		}

		_, err = modelPublicService.GetModelById(owner, uploadedModel.ID, modelPB.View_VIEW_FULL)
		if err == nil {
			makeJSONResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
			return
		}

		readmeFilePath, ensembleFilePath, err := util.Unzip(tmpFile, config.Config.TritonServer.ModelStore, owner, &uploadedModel)
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

		maxBatchSize := 0
		if ensembleFilePath != "" {
			maxBatchSize, err = util.GetMaxBatchSize(ensembleFilePath)
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
		}

		allowedMaxBatchSize := util.GetSupportedBatchSize(uploadedModel.Instances[0].Task)

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

		wfId, err := modelPublicService.CreateModelAsync(owner, &uploadedModel)
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
			makeJSONResponse(w, 500, "Add Model Error", err.Error())
			return
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(201)

		m := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: false, EmitUnpopulated: true}
		b, err := m.Marshal(&modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
			Name: fmt.Sprintf("operations/%s", wfId),
			Done: false,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
			},
		}})
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
func (h *PublicHandler) CreateModelBinaryFileUpload(stream modelPB.ModelPublicService_CreateModelBinaryFileUploadServer) (err error) {
	logger, _ := logger.GetZapLogger()

	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}
	tmpFile, uploadedModel, modelDefID, err := util.SaveFile(stream)
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
	readmeFilePath, ensembleFilePath, err := util.Unzip(tmpFile, config.Config.TritonServer.ModelStore, owner, uploadedModel)
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

	allowedMaxBatchSize := util.GetSupportedBatchSize(uploadedModel.Instances[0].Task)

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

	wfId, err := h.service.CreateModelAsync(owner, uploadedModel)
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, uploadedModel.ID, uploadedModel.Instances[0].ID)
		return err
	}

	err = stream.SendAndClose(&modelPB.CreateModelBinaryFileUploadResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return
}

func createGitHubModel(h *PublicHandler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
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
	var githubInfo *util.GitHubInfo
	if config.Config.Server.ItMode {
		githubInfo = &util.GitHubInfo{
			Description: "This is a test model",
			Visibility:  "public",
			Tags:        []util.Tag{{Name: "v1.0-cpu"}, {Name: "v1.1-cpu"}},
		}
	} else {
		githubInfo, err = util.GetGitHubRepoInfo(modelConfig.Repository)
		if err != nil {
			return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub Info")
		}
		if len(githubInfo.Tags) == 0 {
			return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "There is no tag in GitHub repository")
		}
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
	description := ""
	if req.Model.Description != nil {
		description = *req.Model.Description
	}
	githubModel := datamodel.Model{
		ID:                 req.Model.Id,
		ModelDefinitionUid: modelDefinition.UID,
		Owner:              owner,
		Visibility:         datamodel.ModelVisibility(visibility),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
		Instances:     []datamodel.ModelInstance{},
	}

	for _, tag := range githubInfo.Tags {
		instanceConfig := datamodel.GitHubModelInstanceConfiguration{
			Repository: modelConfig.Repository,
			HtmlUrl:    "https://github.com/" + modelConfig.Repository + "/tree/" + tag.Name,
			Tag:        tag.Name,
		}
		rdid, _ := uuid.NewV4()
		modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String()) + ""
		if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
			modelSrcDir = util.MODEL_CACHE_DIR + "/" + instanceConfig.Repository + instanceConfig.Tag
		}

		if config.Config.Server.ItMode { // use local model for testing to remove internet connection issue while testing
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
			if err := cmd.Run(); err != nil {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, tag.Name)
				return &modelPB.CreateModelResponse{}, err
			}
		} else {
			err = util.GitHubClone(modelSrcDir, instanceConfig, false)
			if err != nil {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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
		}
		bInstanceConfig, _ := json.Marshal(instanceConfig)
		instance := datamodel.ModelInstance{
			ID:            tag.Name,
			State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			Configuration: bInstanceConfig,
		}

		readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, githubModel.ID, &instance)
		if !config.Config.Cache.Model {
			_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
		}
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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
					fmt.Sprintf("[handler] create a model error: %s", err.Error()),
					"README.md file",
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
						fmt.Sprintf("[handler] create a model error: %s", err.Error()),
						"README.md file",
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
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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

		allowedMaxBatchSize := util.GetSupportedBatchSize(instance.Task)

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
	wfId, err := h.service.CreateModelAsync(owner, &githubModel)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func createArtiVCModel(h *PublicHandler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
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
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
		Instances:     []datamodel.ModelInstance{},
	}
	var tags []string
	if !config.Config.Server.ItMode {
		rdid, _ := uuid.NewV4()
		tmpDir := fmt.Sprintf("./%s", rdid.String())
		tags, err = util.ArtiVCGetTags(tmpDir, modelConfig)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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
	} else {
		tags = append(tags, "v1.0-cpu") // use local model for integration test mode
	}
	for _, tag := range tags {
		instanceConfig := datamodel.ArtiVCModelInstanceConfiguration{
			Url: modelConfig.Url,
			Tag: tag,
		}
		rdid, _ := uuid.NewV4()
		modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
		if config.Config.Server.ItMode { // use local model for testing to remove internet connection issue while testing
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
			if err := cmd.Run(); err != nil {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
				return &modelPB.CreateModelResponse{}, err
			}
		} else {
			err = util.ArtiVCClone(modelSrcDir, modelConfig, instanceConfig, false)
			if err != nil {
				st, e := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					fmt.Sprintf("[handler] create a model error: %s", err.Error()),
					"ArtiVC",
					"Clone repository",
					"",
					err.Error(),
				)
				if e != nil {
					logger.Error(e.Error())
				}
				_ = os.RemoveAll(modelSrcDir)
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, tag)
				return &modelPB.CreateModelResponse{}, st.Err()
			}
			util.AddMissingTritonModelFolder(modelSrcDir) // large files not pull then need to create triton model folder
		}
		bInstanceConfig, _ := json.Marshal(instanceConfig)
		instance := datamodel.ModelInstance{
			ID:            tag,
			State:         datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			Configuration: bInstanceConfig,
		}

		readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, artivcModel.ID, &instance)
		_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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
				st, e := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					fmt.Sprintf("[handler] create a model error: %s", err.Error()),
					"README.md file",
					"Could not get meta data from README.md file",
					"",
					err.Error(),
				)
				if e != nil {
					logger.Error(e.Error())
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
						"README.md file",
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
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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

		allowedMaxBatchSize := util.GetSupportedBatchSize(instance.Task)

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
	wfId, err := h.service.CreateModelAsync(owner, &artivcModel)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model service",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
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

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func createHuggingFaceModel(h *PublicHandler, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {
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
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
		Instances:     []datamodel.ModelInstance{},
	}
	rdid, _ := uuid.NewV4()
	configTmpDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if config.Config.Server.ItMode { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/tiny-vit-random/* %s", configTmpDir, configTmpDir))
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(configTmpDir)
			return &modelPB.CreateModelResponse{}, err
		}
	} else {
		if err := util.HuggingFaceClone(configTmpDir, modelConfig); err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"Huggingface",
				"Clone model repository",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			_ = os.RemoveAll(configTmpDir)
			return &modelPB.CreateModelResponse{}, st.Err()
		}
	}
	rdid, _ = uuid.NewV4()
	modelDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if err := util.GenerateHuggingFaceModel(configTmpDir, modelDir, req.Model.Id); err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Huggingface",
			"Generate HuggingFace model",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
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
	readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelDir, config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, &instance)
	_ = os.RemoveAll(modelDir) // remove uploaded temporary files
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model folder structure",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, "latest")
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"README.md file",
				"Could not get meta data from README.md file",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
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
					"README.md file",
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
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
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

	allowedMaxBatchSize := util.GetSupportedBatchSize(instance.Task)

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

	wfId, err := h.service.CreateModelAsync(owner, &huggingfaceModel)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model service",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, "latest")
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) CreateModel(ctx context.Context, req *modelPB.CreateModelRequest) (*modelPB.CreateModelResponse, error) {
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

func (h *PublicHandler) ListModels(ctx context.Context, req *modelPB.ListModelsRequest) (*modelPB.ListModelsResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.ListModelsResponse{}, err
	}
	dbModels, nextPageToken, totalSize, err := h.service.ListModels(owner, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelsResponse{}, err
	}

	pbModels := []*modelPB.Model{}
	for _, dbModel := range dbModels {
		modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.ListModelsResponse{}, err
		}
		pbModels = append(pbModels, DBModelToPBModel(&modelDef, &dbModel))
	}

	resp := modelPB.ListModelsResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) LookUpModel(ctx context.Context, req *modelPB.LookUpModelRequest) (*modelPB.LookUpModelResponse, error) {
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

func (h *PublicHandler) GetModel(ctx context.Context, req *modelPB.GetModelRequest) (*modelPB.GetModelResponse, error) {
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

func (h *PublicHandler) UpdateModel(ctx context.Context, req *modelPB.UpdateModelRequest) (*modelPB.UpdateModelResponse, error) {
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
				updateModel.Description = sql.NullString{
					String: *req.Model.Description,
					Valid:  true,
				}
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

func (h *PublicHandler) DeleteModel(ctx context.Context, req *modelPB.DeleteModelRequest) (*modelPB.DeleteModelResponse, error) {
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

func (h *PublicHandler) RenameModel(ctx context.Context, req *modelPB.RenameModelRequest) (*modelPB.RenameModelResponse, error) {
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

func (h *PublicHandler) PublishModel(ctx context.Context, req *modelPB.PublishModelRequest) (*modelPB.PublishModelResponse, error) {
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

func (h *PublicHandler) UnpublishModel(ctx context.Context, req *modelPB.UnpublishModelRequest) (*modelPB.UnpublishModelResponse, error) {
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

func (h *PublicHandler) GetModelInstance(ctx context.Context, req *modelPB.GetModelInstanceRequest) (*modelPB.GetModelInstanceResponse, error) {
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

func (h *PublicHandler) LookUpModelInstance(ctx context.Context, req *modelPB.LookUpModelInstanceRequest) (*modelPB.LookUpModelInstanceResponse, error) {
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

func (h *PublicHandler) ListModelInstances(ctx context.Context, req *modelPB.ListModelInstancesRequest) (*modelPB.ListModelInstancesResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.ListModelInstancesResponse{}, err
	}

	modelID, err := resource.GetID(req.Parent)
	if err != nil {
		return &modelPB.ListModelInstancesResponse{}, err
	}
	modelInDB, err := h.service.GetModelById(owner, modelID, req.GetView())
	if err != nil {
		return &modelPB.ListModelInstancesResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(modelInDB.ModelDefinitionUid)
	if err != nil {
		return &modelPB.ListModelInstancesResponse{}, err
	}

	dbModelInstances, nextPageToken, totalSize, err := h.service.ListModelInstances(modelInDB.UID, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelInstancesResponse{}, err
	}

	pbInstances := []*modelPB.ModelInstance{}
	for _, dbModelInstance := range dbModelInstances {
		pbInstances = append(pbInstances, DBModelInstanceToPBModelInstance(&modelDef, &modelInDB, &dbModelInstance))
	}

	resp := modelPB.ListModelInstancesResponse{
		Instances:     pbInstances,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) DeployModelInstance(ctx context.Context, req *modelPB.DeployModelInstanceRequest) (*modelPB.DeployModelInstanceResponse, error) {
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

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	if dbModelInstance.State != datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE) {
		return &modelPB.DeployModelInstanceResponse{},
			status.Error(codes.FailedPrecondition, fmt.Sprintf("Deploy model only work with offline model instance state, current model state is %s",
				modelPB.ModelInstance_State_name[int32(dbModelInstance.State)]))
	}

	_, err = h.service.GetTritonModels(dbModelInstance.UID)
	if err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	// temporary change state to STATE_UNSPECIFIED during deploying the model
	// the state will be changed after deploying to STATE_ONLINE or STATE_ERROR
	if err := h.service.UpdateModelInstance(dbModelInstance.UID, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_UNSPECIFIED),
	}); err != nil {
		return &modelPB.DeployModelInstanceResponse{}, err
	}

	wfId, err := h.service.DeployModelInstanceAsync(owner, dbModel.UID, dbModelInstance.UID)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] deploy a model error: %s", err.Error()),
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

	return &modelPB.DeployModelInstanceResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) UndeployModelInstance(ctx context.Context, req *modelPB.UndeployModelInstanceRequest) (*modelPB.UndeployModelInstanceResponse, error) {

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

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	if dbModelInstance.State != datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ONLINE) {
		return &modelPB.UndeployModelInstanceResponse{},
			status.Error(codes.FailedPrecondition, fmt.Sprintf("undeploy model only work with online model instance state, current model state is %s",
				modelPB.ModelInstance_State_name[int32(dbModelInstance.State)]))
	}

	// temporary change state to STATE_UNSPECIFIED during undeploying the model
	// the state will be changed after undeploying to STATE_OFFLINE or STATE_ERROR
	if err := h.service.UpdateModelInstance(dbModelInstance.UID, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_UNSPECIFIED),
	}); err != nil {
		return &modelPB.UndeployModelInstanceResponse{}, err
	}
	wfId, err := h.service.UndeployModelInstanceAsync(owner, dbModel.UID, dbModelInstance.UID)
	if err != nil {
		// Manually set the custom header to have a StatusUnprocessableEntity http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusUnprocessableEntity))); err != nil {
			return &modelPB.UndeployModelInstanceResponse{}, status.Errorf(codes.Internal, err.Error())
		}
		return &modelPB.UndeployModelInstanceResponse{}, err
	}

	return &modelPB.UndeployModelInstanceResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) TestModelInstanceBinaryFileUpload(stream modelPB.ModelPublicService_TestModelInstanceBinaryFileUploadServer) error {
	logger, _ := logger.GetZapLogger()
	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}

	triggerInput, modelID, instanceID, err := savePredictInputsTestMode(stream)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	modelInDB, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}
	modelInstanceInDB, err := h.service.GetModelInstance(modelInDB.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	numberOfInferences := 1
	switch modelPB.ModelInstance_Task(modelInstanceInDB.Task) {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_KEYPOINT:
		numberOfInferences = len(triggerInput.([][]byte))
	}

	// check whether model support batching or not. If not, raise an error
	if numberOfInferences > 1 {
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
	response, err := h.service.ModelInferTestMode(owner, modelInstanceInDB.UID, triggerInput, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
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
		Task:        task,
		TaskOutputs: response,
	})
	return err
}

func (h *PublicHandler) TriggerModelInstanceBinaryFileUpload(stream modelPB.ModelPublicService_TriggerModelInstanceBinaryFileUploadServer) error {
	logger, _ := logger.GetZapLogger()
	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}
	triggerInput, modelID, instanceID, err := savePredictInputsTriggerMode(stream)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
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
	numberOfInferences := 1
	switch modelPB.ModelInstance_Task(modelInstanceInDB.Task) {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_KEYPOINT:
		numberOfInferences = len(triggerInput.([][]byte))
	}
	if numberOfInferences > 1 {
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
	response, err := h.service.ModelInfer(modelInstanceInDB.UID, triggerInput, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
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
		Task:        task,
		TaskOutputs: response,
	})
	return err
}

func (h *PublicHandler) TriggerModelInstance(ctx context.Context, req *modelPB.TriggerModelInstanceRequest) (*modelPB.TriggerModelInstanceResponse, error) {
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
	var inputInfer interface{}
	var lenInputs = 1
	switch modelPB.ModelInstance_Task(modelInstanceInDB.Task) {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_KEYPOINT,
		modelPB.ModelInstance_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(req)
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(req)
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textToImage
	case modelPB.ModelInstance_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(req)
		if err != nil {
			return &modelPB.TriggerModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
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
	response, err := h.service.ModelInfer(modelInstanceInDB.UID, inputInfer, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
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
		Task:        task,
		TaskOutputs: response,
	}, nil
}

func (h *PublicHandler) TestModelInstance(ctx context.Context, req *modelPB.TestModelInstanceRequest) (*modelPB.TestModelInstanceResponse, error) {
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

	var inputInfer interface{}
	var lenInputs = 1
	switch modelPB.ModelInstance_Task(modelInstanceInDB.Task) {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_KEYPOINT,
		modelPB.ModelInstance_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(&modelPB.TriggerModelInstanceRequest{
			Name:       req.Name,
			TaskInputs: req.TaskInputs,
		})
		if err != nil {
			return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(&modelPB.TriggerModelInstanceRequest{
			Name:       req.Name,
			TaskInputs: req.TaskInputs,
		})
		if err != nil {
			return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textToImage
	case modelPB.ModelInstance_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(
			&modelPB.TriggerModelInstanceRequest{
				Name:       req.Name,
				TaskInputs: req.TaskInputs,
			})
		if err != nil {
			return &modelPB.TestModelInstanceResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGeneration
	}

	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
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
	response, err := h.service.ModelInferTestMode(owner, modelInstanceInDB.UID, inputInfer, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
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
		Task:        task,
		TaskOutputs: response,
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
		pipelinePublicServiceClient, pipelinePublicServiceClientConn := external.InitPipelinePublicServiceClient()
		defer pipelinePublicServiceClientConn.Close()
		redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
		defer redisClient.Close()
		temporalClient, err := client.Dial(client.Options{
			// ZapAdapter implements log.Logger interface and can be passed
			// to the client constructor using client using client.Options.
			Logger:    zapadapter.NewZapAdapter(logger),
			HostPort:  config.Config.Temporal.ClientOptions.HostPort,
			Namespace: modelWorker.Namespace,
		})
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer temporalClient.Close()
		modelPublicService := service.NewService(modelRepository, tritonService, pipelinePublicServiceClient, redisClient, temporalClient)

		modelID, instanceID, err := resource.GetModelInstanceID(instanceName)
		if err != nil {
			makeJSONResponse(w, 400, "Parameter invalid", "Required parameter instance_name is invalid")
			return
		}

		modelInDB, err := modelPublicService.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
		if err != nil {
			makeJSONResponse(w, 404, "Model not found", "The model not found in server")
			return
		}

		modelInstanceInDB, err := modelPublicService.GetModelInstance(modelInDB.UID, instanceID, modelPB.View_VIEW_FULL)
		if err != nil {
			makeJSONResponse(w, 404, "Model instance not found", "The model instance not found in server")
			return
		}

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeJSONResponse(w, 400, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
			return
		}

		var inputInfer interface{}
		var lenInputs = 1
		switch modelPB.ModelInstance_Task(modelInstanceInDB.Task) {
		case modelPB.ModelInstance_TASK_CLASSIFICATION,
			modelPB.ModelInstance_TASK_DETECTION,
			modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
			modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
			modelPB.ModelInstance_TASK_OCR,
			modelPB.ModelInstance_TASK_KEYPOINT,
			modelPB.ModelInstance_TASK_UNSPECIFIED:
			imageInput, err := parseImageFormDataInputsToBytes(r)
			if err != nil {
				makeJSONResponse(w, 400, "File Input Error", err.Error())
				return
			}
			lenInputs = len(imageInput)
			inputInfer = imageInput
		case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE:
			textToImage, err := parseImageFormDataTextToImageInputs(r)
			if err != nil {
				makeJSONResponse(w, 400, "Parser input error", err.Error())
				return
			}
			lenInputs = 1
			inputInfer = textToImage
		case modelPB.ModelInstance_TASK_TEXT_GENERATION:
			textGeneration, err := parseTextFormDataTextGenerationInputs(r)
			if err != nil {
				makeJSONResponse(w, 400, "Parser input error", err.Error())
				return
			}
			lenInputs = 1
			inputInfer = textGeneration
		}

		// check whether model support batching or not. If not, raise an error
		if lenInputs > 1 {
			tritonModelInDB, err := modelPublicService.GetTritonEnsembleModel(modelInstanceInDB.UID)
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
		var response []*modelPB.TaskOutput
		if mode == "test" {
			response, err = modelPublicService.ModelInferTestMode(owner, modelInstanceInDB.UID, inputInfer, task)
		} else {
			response, err = modelPublicService.ModelInfer(modelInstanceInDB.UID, inputInfer, task)
		}
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] inference model error: %s", err.Error()),
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
			Task:        task,
			TaskOutputs: response,
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

func (h *PublicHandler) GetModelInstanceCard(ctx context.Context, req *modelPB.GetModelInstanceCardRequest) (*modelPB.GetModelInstanceCardResponse, error) {
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

	content, _ := os.ReadFile(readmeFilePath)

	return &modelPB.GetModelInstanceCardResponse{Readme: &modelPB.ModelInstanceCard{
		Name:     req.Name,
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Content:  []byte(content),
	}}, nil
}

func (h *PublicHandler) GetModelDefinition(ctx context.Context, req *modelPB.GetModelDefinitionRequest) (*modelPB.GetModelDefinitionResponse, error) {
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

func (h *PublicHandler) ListModelDefinitions(ctx context.Context, req *modelPB.ListModelDefinitionsRequest) (*modelPB.ListModelDefinitionsResponse, error) {
	dbModelDefinitions, nextPageToken, totalSize, err := h.service.ListModelDefinitions(req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelDefinitionsResponse{}, err
	}

	pbDefinitions := []*modelPB.ModelDefinition{}
	for _, dbModelDefinition := range dbModelDefinitions {
		pbDefinitions = append(pbDefinitions, DBModelDefinitionToPBModelDefinition(&dbModelDefinition))
	}

	resp := modelPB.ListModelDefinitionsResponse{
		ModelDefinitions: pbDefinitions,
		NextPageToken:    nextPageToken,
		TotalSize:        totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) GetModelOperation(ctx context.Context, req *modelPB.GetModelOperationRequest) (*modelPB.GetModelOperationResponse, error) {
	operationId, err := resource.GetOperationID(req.Name)
	fmt.Println("operationId", operationId)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}
	operation, modelInstanceParam, operationType, err := h.service.GetOperation(operationId)
	fmt.Println("operation, modelInstanceParam, operationType, err", operation, modelInstanceParam, operationType, err)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}

	if !operation.Done {
		return &modelPB.GetModelOperationResponse{
			Operation: operation,
		}, nil
	}

	dbModel, err := h.service.GetModelByUid(modelInstanceParam.Owner, modelInstanceParam.ModelUID, req.GetView())
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}
	switch operationType {
	case string(util.OperationTypeCreate):
		pbModel := DBModelToPBModel(&modelDef, &dbModel)
		res, err := anypb.New(pbModel)
		if err != nil {
			return &modelPB.GetModelOperationResponse{}, err
		}

		operation.Result = &longrunningpb.Operation_Response{
			Response: res,
		}
		return &modelPB.GetModelOperationResponse{
			Operation: operation,
		}, nil
	case string(util.OperationTypeDeploy), string(util.OperationTypeUnDeploy):
		dbModelInstance, err := h.service.GetModelInstanceByUid(modelInstanceParam.ModelUID, modelInstanceParam.ModelInstanceUID, req.GetView())
		if err != nil {
			return &modelPB.GetModelOperationResponse{}, err
		}
		pbModelInstance := DBModelInstanceToPBModelInstance(&modelDef, &dbModel, &dbModelInstance)

		res, err := anypb.New(pbModelInstance)
		if err != nil {
			return &modelPB.GetModelOperationResponse{}, err
		}

		operation.Result = &longrunningpb.Operation_Response{
			Response: res,
		}
		return &modelPB.GetModelOperationResponse{
			Operation: operation,
		}, nil
	default:
		return &modelPB.GetModelOperationResponse{}, fmt.Errorf("operation type not supported")
	}
}

func (h *PublicHandler) ListModelOperations(ctx context.Context, req *modelPB.ListModelOperationsRequest) (*modelPB.ListModelOperationsResponse, error) {
	pageSize := util.DefaultPageSize
	if req.PageSize != nil {
		pageSize = int(*req.PageSize)
	}
	operations, _, nextPageToken, totalSize, err := h.service.ListOperation(pageSize, req.PageToken)
	if err != nil {
		return &modelPB.ListModelOperationsResponse{}, err
	}

	return &modelPB.ListModelOperationsResponse{
		Operations:    operations,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

func (h *PublicHandler) CancelModelOperation(ctx context.Context, req *modelPB.CancelModelOperationRequest) (*modelPB.CancelModelOperationResponse, error) {
	operationId, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &modelPB.CancelModelOperationResponse{}, err
	}

	_, modelInstanceParam, operationType, err := h.service.GetOperation(operationId)
	if err != nil {
		return &modelPB.CancelModelOperationResponse{}, err
	}

	// get model instance state before cancel operation to set in case of operation in progess and state is UNSPECIFIED
	dbModelInstance, err := h.service.GetModelInstanceByUid(modelInstanceParam.ModelUID, modelInstanceParam.ModelInstanceUID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return &modelPB.CancelModelOperationResponse{}, err
	}

	if err = h.service.CancelOperation(operationId); err != nil {
		return &modelPB.CancelModelOperationResponse{}, err
	}

	// Fix for corner case: maybe when cancel operation in Temporal, the Temporal workflow already trigger Triton server to deploy/undeploy model instance
	switch operationType {
	case string(util.OperationTypeDeploy):
		if dbModelInstance.State == datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_UNSPECIFIED) {
			if err := h.service.UpdateModelInstance(dbModelInstance.UID, datamodel.ModelInstance{
				State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
			}); err != nil {
				return &modelPB.CancelModelOperationResponse{}, err
			}
		}
	case string(util.OperationTypeUnDeploy):
		if dbModelInstance.State == datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_UNSPECIFIED) {
			if err := h.service.UpdateModelInstance(dbModelInstance.UID, datamodel.ModelInstance{
				State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ONLINE),
			}); err != nil {
				return &modelPB.CancelModelOperationResponse{}, err
			}
		}
	}

	return &modelPB.CancelModelOperationResponse{}, nil
}
