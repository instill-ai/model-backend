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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/util"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"Id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"Name", "Uid", "Visibility", "Owner", "CreateTime", "UpdateTime"}

var tracer = otel.Tracer("model-backend.public-handler.tracer")

type PublicHandler struct {
	modelPB.UnimplementedModelPublicServiceServer
	service service.Service
	triton  triton.Triton
}

func NewPublicHandler(ctx context.Context, s service.Service, t triton.Triton) modelPB.ModelPublicServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PublicHandler{
		service: s,
		triton:  t,
	}
}

func savePredictInputsTriggerMode(stream modelPB.ModelPublicService_TriggerModelBinaryFileUploadServer) (triggerInput interface{}, modelID string, err error) {

	var firstChunk = true

	var fileData *modelPB.TriggerModelBinaryFileUploadRequest

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
			return nil, "", err
		}

		if firstChunk { //first chunk contains model instance name
			firstChunk = false
			modelID, err = resource.GetModelID(fileData.Name) // format "models/{model}"
			if err != nil {
				return nil, "", err
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
				return nil, "", fmt.Errorf("unsupported task input type")
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
				return nil, "", fmt.Errorf("unsupported task input type")
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
			return nil, "", fmt.Errorf("wrong parameter length of files")
		}
		imageBytes := make([][]byte, len(fileLengths))
		start := uint64(0)
		for i := 0; i < len(fileLengths); i++ {
			buff := new(bytes.Buffer)
			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
			if err != nil {
				return nil, "", err
			}
			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
			if err != nil {
				return nil, "", err
			}
			imageBytes[i] = buff.Bytes()
			start += fileLengths[i]
		}
		return imageBytes, modelID, nil
	case *modelPB.TaskInputStream_TextToImage:
		return textToImageInput, modelID, nil
	case *modelPB.TaskInputStream_TextGeneration:
		return textGeneration, modelID, nil
	}
	return nil, "", fmt.Errorf("unsupported task input type")
}

func savePredictInputsTestMode(stream modelPB.ModelPublicService_TestModelBinaryFileUploadServer) (triggerInput interface{}, modelID string, err error) {
	var firstChunk = true
	var fileData *modelPB.TestModelBinaryFileUploadRequest

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
			return nil, "", err
		}

		if firstChunk { //first chunk contains file name
			firstChunk = false
			modelID, err = resource.GetModelID(fileData.Name) // format "models/{model}"
			if err != nil {
				return nil, "", err
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
				return nil, "", fmt.Errorf("unsupported task input type")
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
			return nil, "", fmt.Errorf("unsupported task input type")
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
			return nil, "", fmt.Errorf("wrong parameter length of files")
		}
		imageBytes := make([][]byte, len(fileLengths))
		start := uint64(0)
		for i := 0; i < len(fileLengths); i++ {
			buff := new(bytes.Buffer)
			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
			if err != nil {
				return nil, "", err
			}
			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
			if err != nil {
				return nil, "", err
			}
			imageBytes[i] = buff.Bytes()
			start += fileLengths[i]
		}
		return imageBytes, modelID, nil
	case *modelPB.TaskInputStream_TextToImage:
		return textToImageInput, modelID, nil
	case *modelPB.TaskInputStream_TextGeneration:
		return textGeneration, modelID, nil
	}
	return nil, "", fmt.Errorf("unsupported task input type")

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
	if !h.triton.IsTritonServerReady(ctx) {
		return &modelPB.LivenessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !h.triton.IsTritonServerReady(ctx) {
		return &modelPB.ReadinessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelPB.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

// HandleCreateModelByMultiPartFormData is a custom handler
func HandleCreateModelByMultiPartFormData(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	eventName := "HandleCreateModelByMultiPartFormData"

	ctx, span := tracer.Start(req.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		span.SetStatus(1, "")
		return
	}

	owner, err := resource.GetOwnerCustom(req, s.GetMgmtPrivateServiceClient(), s.GetRedisClient())
	if err != nil {
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			span.SetStatus(1, "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'jwt-sub' or 'owner-id' not found in your header")
			span.SetStatus(1, "Required parameter 'jwt-sub' or 'owner-id' not found in your header")
			return
		}
	}
	ownerPermalink := GenOwnerPermalink(owner)

	modelID := req.FormValue("id")
	if modelID == "" {
		makeJSONResponse(w, 400, "Missing parameter", "Model Id need to be specified")
		span.SetStatus(1, "Model Id need to be specified")
		return
	}

	modelDefinitionName := req.FormValue("model_definition")
	if modelDefinitionName == "" {
		makeJSONResponse(w, 400, "Missing parameter", "modelDefinitionName need to be specified")
		span.SetStatus(1, "modelDefinitionName need to be specified")
		return
	}
	modelDefinitionID, err := resource.GetDefinitionID(modelDefinitionName)
	if err != nil {
		makeJSONResponse(w, 400, "Invalid parameter", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	viz := req.FormValue("visibility")
	var visibility modelPB.Model_Visibility
	if viz != "" {
		if util.Visibility[viz] == modelPB.Model_VISIBILITY_UNSPECIFIED {
			makeJSONResponse(w, 400, "Invalid parameter", "Visibility is invalid")
			span.SetStatus(1, "Visibility is invalid")
			return
		} else {
			visibility = util.Visibility[viz]
		}
	} else {
		visibility = modelPB.Model_VISIBILITY_PRIVATE
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		return
	}
	file, fileHeader, err := req.FormFile("content")
	if err != nil {
		makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
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
		span.SetStatus(1, "Error reading input file")
		return
	}
	rdid, _ := uuid.NewV4()
	tmpFile := path.Join("/tmp", rdid.String())
	fp, err := os.Create(tmpFile)
	if err != nil {
		makeJSONResponse(w, 400, "File Error", "Error reading input file")
		span.SetStatus(1, "Error reading input file")
		return
	}
	err = util.WriteToFp(fp, buf.Bytes())
	if err != nil {
		makeJSONResponse(w, 400, "File Error", "Error reading input file")
		span.SetStatus(1, "Error reading input file")
		return
	}

	// validate model configuration
	localModelDefinition, err := s.GetRepository().GetModelDefinition(modelDefinitionID)
	if err != nil {
		makeJSONResponse(w, 400, "Parameter invalid", "ModelDefinitionId not found")
		span.SetStatus(1, "ModelDefinitionId not found")
		return
	}
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(localModelDefinition.ModelSpec.String()), rs); err != nil {
		makeJSONResponse(w, 500, "Add Model Error", "Could not get model definition")
		span.SetStatus(1, "Could not get model definition")
		return
	}
	modelConfiguration := datamodel.LocalModelConfiguration{
		Content: fileHeader.Filename,
	}

	if err := datamodel.ValidateJSONSchema(rs, modelConfiguration, true); err != nil {
		makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		span.SetStatus(1, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		return
	}
	modelConfiguration.Tag = "latest" // Set after validation. Because the model definition do not contain tag.

	bModelConfig, _ := json.Marshal(modelConfiguration)
	var uploadedModel = datamodel.Model{
		ID:                 modelID,
		ModelDefinitionUid: localModelDefinition.UID,
		Owner:              ownerPermalink,
		Visibility:         datamodel.ModelVisibility(visibility),
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: req.FormValue("description"),
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, DBModelToPBModel(ctx, &localModelDefinition, &uploadedModel, ownerPermalink), true); err != nil {
		makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model definition is invalid %v", err.Error()))
		span.SetStatus(1, fmt.Sprintf("Model definition is invalid %v", err.Error()))
		return
	}

	_, err = s.GetModelByID(req.Context(), ownerPermalink, uploadedModel.ID, modelPB.View_VIEW_FULL)
	if err == nil {
		makeJSONResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
		span.SetStatus(1, fmt.Sprintf("The model %v already existed", uploadedModel.ID))
		return
	}

	readmeFilePath, ensembleFilePath, err := util.Unzip(tmpFile, config.Config.TritonServer.ModelStore, ownerPermalink, &uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 400, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
		return
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
			makeJSONResponse(w, 400, "Add Model Error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		if modelMeta.Task == "" {
			uploadedModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
		} else {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Task = datamodel.ModelTask(val)
			} else {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
				makeJSONResponse(w, 400, "Add Model Error", "README.md contains unsupported task")
				span.SetStatus(1, "README.md contains unsupported task")
				return
			}
		}
	} else {
		uploadedModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
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
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
			span.SetStatus(1, err.Error())
			return
		}
	}

	allowedMaxBatchSize := util.GetSupportedBatchSize(uploadedModel.Task)

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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
		obj, _ := json.Marshal(st.Details())
		makeJSONResponse(w, 400, st.Message(), string(obj))
		span.SetStatus(1, string(obj))
		return
	}

	wfId, err := s.CreateModelAsync(req.Context(), ownerPermalink, &uploadedModel)
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 500, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 500, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(uploadedModel),
	)))

	_, _ = w.Write(b)

}

// AddModel - upload a model to the model server
func (h *PublicHandler) CreateModelBinaryFileUpload(stream modelPB.ModelPublicService_CreateModelBinaryFileUploadServer) (err error) {

	eventName := "CreateModelBinaryFileUpload"

	ctx, span := tracer.Start(stream.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(stream.Context(), h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	tmpFile, uploadedModel, modelDefID, err := util.SaveFile(stream)
	if err != nil {
		span.SetStatus(1, err.Error())
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	_, err = h.service.GetModelByID(stream.Context(), ownerPermalink, uploadedModel.ID, modelPB.View_VIEW_FULL)
	if err == nil {
		span.SetStatus(1, fmt.Sprintf("The model %v already existed", uploadedModel.ID))
		return status.Errorf(codes.AlreadyExists, fmt.Sprintf("The model %v already existed", uploadedModel.ID))
	}

	modelDef, err := h.service.GetModelDefinition(stream.Context(), modelDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	uploadedModel.ModelDefinitionUid = modelDef.UID

	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, DBModelToPBModel(ctx, &modelDef, uploadedModel, ownerPermalink), true); err != nil {
		span.SetStatus(1, err.Error())
		return status.Errorf(codes.InvalidArgument, err.Error())
	}

	uploadedModel.Owner = ownerPermalink

	// extract zip file from tmp to models directory
	readmeFilePath, ensembleFilePath, err := util.Unzip(tmpFile, config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
		span.SetStatus(1, err.Error())
		return status.Errorf(codes.Internal, err.Error())
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := util.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
			span.SetStatus(1, err.Error())
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		if modelMeta.Task == "" {
			uploadedModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
		} else {
			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Task = datamodel.ModelTask(val)
			} else {
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
				span.SetStatus(1, "README.md contains unsupported task")
				return status.Errorf(codes.InvalidArgument, "README.md contains unsupported task")
			}
		}
	} else {
		uploadedModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
		span.SetStatus(1, err.Error())
		return st.Err()
	}

	allowedMaxBatchSize := util.GetSupportedBatchSize(uploadedModel.Task)

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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
		span.SetStatus(1, st.String())
		return st.Err()
	}

	wfId, err := h.service.CreateModelAsync(stream.Context(), ownerPermalink, uploadedModel)
	if err != nil {
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, uploadedModel.ID, "latest")
		span.SetStatus(1, err.Error())
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
		span.SetStatus(1, err.Error())
		return status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(uploadedModel),
	)))

	return
}

func createGitHubModel(service service.Service, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {

	eventName := "CreateGitHubModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var modelConfig datamodel.GitHubModelConfiguration
	b, err := req.Model.Configuration.MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.Repository == "" {
		span.SetStatus(1, "Invalid GitHub URL")
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub URL")
	}
	var githubInfo *util.GitHubInfo
	if config.Config.Server.ItMode.Enabled {
		githubInfo = &util.GitHubInfo{
			Description: "This is a test model",
			Visibility:  "public",
			Tags:        []util.Tag{{Name: "v1.0-cpu"}, {Name: "v1.1-cpu"}},
		}
	} else {
		githubInfo, err = util.GetGitHubRepoInfo(modelConfig.Repository)
		if err != nil {
			span.SetStatus(1, "Invalid GitHub Info")
			return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub Info")
		}
		if len(githubInfo.Tags) == 0 {
			span.SetStatus(1, "There is no tag in GitHub repository")
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
		Tag:        modelConfig.Tag,
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
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String()) + ""
	if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
		modelSrcDir = util.MODEL_CACHE_DIR + "/" + modelConfig.Repository + modelConfig.Tag
	}

	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
		if err := cmd.Run(); err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &modelPB.CreateModelResponse{}, err
		}
	} else {
		err = util.GitHubClone(modelSrcDir, modelConfig, false)
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
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &modelPB.CreateModelResponse{}, st.Err()
		}
	}
	readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, &githubModel)
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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
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
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			githubModel.Task = datamodel.ModelTask(val)
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
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, githubModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &modelPB.CreateModelResponse{}, st.Err()
			} else {
				githubModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
			}
		}
	} else {
		githubModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	allowedMaxBatchSize := util.GetSupportedBatchSize(githubModel.Task)

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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	wfId, err := service.CreateModelAsync(ctx, owner, &githubModel)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	user, err := resource.GetOwner(ctx, service.GetMgmtPrivateServiceClient(), service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		user,
		eventName,
		custom_otel.SetEventResource(githubModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfId),
			},
		}),
	)))

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func createHuggingFaceModel(service service.Service, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {

	eventName := "CreateHuggingFaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var modelConfig datamodel.HuggingFaceModelConfiguration
	b, err := req.Model.GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.RepoId == "" {
		span.SetStatus(1, "Invalid model ID")
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Invalid model ID")
	}
	modelConfig.HtmlUrl = "https://huggingface.co/" + modelConfig.RepoId
	modelConfig.Tag = "latest"

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
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}
	rdid, _ := uuid.NewV4()
	configTmpDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/tiny-vit-random/* %s", configTmpDir, configTmpDir))
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(configTmpDir)
			span.SetStatus(1, err.Error())
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
			span.SetStatus(1, err.Error())
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}
	_ = os.RemoveAll(configTmpDir)

	readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelDir, config.Config.TritonServer.ModelStore, owner, &huggingfaceModel)

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
		span.SetStatus(1, st.Err().Error())
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
			span.SetStatus(1, st.Err().Error())
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		if modelMeta.Task != "" {

			if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				huggingfaceModel.Task = datamodel.ModelTask(val)
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
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, huggingfaceModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &modelPB.CreateModelResponse{}, st.Err()
			}
		} else {
			if len(modelMeta.Tags) == 0 {
				huggingfaceModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
			} else { // check in tags also for HuggingFace model card README.md
				huggingfaceModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
				for _, tag := range modelMeta.Tags {
					if val, ok := util.Tags[strings.ToUpper(tag)]; ok {
						huggingfaceModel.Task = datamodel.ModelTask(val)
						break
					}
				}
			}
		}
	} else {
		huggingfaceModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	allowedMaxBatchSize := util.GetSupportedBatchSize(huggingfaceModel.Task)

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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	wfId, err := service.CreateModelAsync(ctx, owner, &huggingfaceModel)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	user, err := resource.GetOwner(ctx, service.GetMgmtPrivateServiceClient(), service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		user,
		eventName,
		custom_otel.SetEventResource(huggingfaceModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfId),
			},
		}),
	)))

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func createArtiVCModel(service service.Service, ctx context.Context, req *modelPB.CreateModelRequest, owner string, modelDefinition *datamodel.ModelDefinition) (*modelPB.CreateModelResponse, error) {

	eventName := "CreateArtiVCModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var modelConfig datamodel.ArtiVCModelConfiguration
	b, err := req.Model.GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.Url == "" {
		span.SetStatus(1, "Invalid GitHub URL")
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
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
		if err := cmd.Run(); err != nil {
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &modelPB.CreateModelResponse{}, err
		}
	} else {
		err = util.ArtiVCClone(modelSrcDir, modelConfig, false)
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
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		util.AddMissingTritonModelFolder(ctx, modelSrcDir) // large files not pull then need to create triton model folder
	}

	readmeFilePath, ensembleFilePath, err := util.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, owner, &artivcModel)
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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
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
			util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &modelPB.CreateModelResponse{}, st.Err()
		}
		if val, ok := util.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			artivcModel.Task = datamodel.ModelTask(val)
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
				util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &modelPB.CreateModelResponse{}, st.Err()
			} else {
				artivcModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
			}
		}
	} else {
		artivcModel.Task = datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	allowedMaxBatchSize := util.GetSupportedBatchSize(artivcModel.Task)

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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	wfId, err := service.CreateModelAsync(ctx, owner, &artivcModel)
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
		util.RemoveModelRepository(config.Config.TritonServer.ModelStore, owner, artivcModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
		return &modelPB.CreateModelResponse{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	user, err := resource.GetOwner(ctx, service.GetMgmtPrivateServiceClient(), service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		user,
		eventName,
		custom_otel.SetEventResource(artivcModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfId),
			},
		}),
	)))

	return &modelPB.CreateModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) CreateModel(ctx context.Context, req *modelPB.CreateModelRequest) (*modelPB.CreateModelResponse, error) {

	ctx, span := tracer.Start(ctx, "CreateModel",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	resp := &modelPB.CreateModelResponse{}
	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	// Set all OUTPUT_ONLY fields to zero value on the requested payload model resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Model, outputOnlyFields); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Return error if REQUIRED fields are not provided in the requested payload model resource
	if err := checkfield.CheckRequiredFields(req.Model, requiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.Model.GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, req.Model, false); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if model, err := h.service.GetModelByID(ctx, ownerPermalink, req.Model.Id, modelPB.View_VIEW_FULL); err == nil {
		if util.HasModelInModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, model.ID) {
			span.SetStatus(1, "Model already existed")
			return resp, status.Errorf(codes.AlreadyExists, "Model already existed")
		}
	}

	if req.Model.Configuration == nil {
		span.SetStatus(1, "Missing Configuration")
		return resp, status.Errorf(codes.InvalidArgument, "Missing Configuration")
	}

	modelDefinitionID, err := resource.GetDefinitionID(req.Model.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	modelDefinition, err := h.service.GetModelDefinition(ctx, modelDefinitionID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// validate model configuration
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(modelDefinition.ModelSpec.String()), rs); err != nil {
		span.SetStatus(1, "Could not get model definition")
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, "Could not get model definition")
	}
	if err := datamodel.ValidateJSONSchema(rs, req.Model.GetConfiguration(), true); err != nil {
		span.SetStatus(1, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		return &modelPB.CreateModelResponse{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
	}

	switch modelDefinitionID {
	case "github":
		return createGitHubModel(h.service, ctx, req, ownerPermalink, &modelDefinition)
	case "artivc":
		return createArtiVCModel(h.service, ctx, req, ownerPermalink, &modelDefinition)
	case "huggingface":
		return createHuggingFaceModel(h.service, ctx, req, ownerPermalink, &modelDefinition)
	default:
		span.SetStatus(1, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
		return resp, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
	}

}

func (h *PublicHandler) ListModels(ctx context.Context, req *modelPB.ListModelsRequest) (*modelPB.ListModelsResponse, error) {

	eventName := "ListModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.ListModelsResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	dbModels, nextPageToken, totalSize, err := h.service.ListModels(ctx, ownerPermalink, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.ListModelsResponse{}, err
	}

	pbModels := []*modelPB.Model{}
	for _, dbModel := range dbModels {
		modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.ListModelsResponse{}, err
		}
		pbModels = append(pbModels, DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink))
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModels),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	resp := modelPB.ListModelsResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) LookUpModel(ctx context.Context, req *modelPB.LookUpModelRequest) (*modelPB.LookUpModelResponse, error) {

	eventName := "LookUpModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	sUID, err := resource.GetID(req.Permalink)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbModel, err := h.service.GetModelByUID(ctx, ownerPermalink, uid, req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.LookUpModelResponse{Model: pbModel}, nil
}

func (h *PublicHandler) GetModel(ctx context.Context, req *modelPB.GetModelRequest) (*modelPB.GetModelResponse, error) {

	eventName := "GetModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	id, err := resource.GetID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelResponse{}, err
	}
	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, id, req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.GetModelResponse{Model: pbModel}, err
}

func (h *PublicHandler) UpdateModel(ctx context.Context, req *modelPB.UpdateModelRequest) (*modelPB.UpdateModelResponse, error) {

	eventName := "UpdateModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UpdateModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	id, err := resource.GetID(req.Model.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UpdateModelResponse{}, err
	}
	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, id, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UpdateModelResponse{}, err
	}

	if ownerPermalink != dbModel.Owner {
		return &modelPB.UpdateModelResponse{}, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	updateModel := datamodel.Model{
		ID:    id,
		State: dbModel.State,
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
	dbModel, err = h.service.UpdateModel(ctx, dbModel.UID, &updateModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UpdateModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UpdateModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.UpdateModelResponse{Model: pbModel}, err
}

func (h *PublicHandler) DeleteModel(ctx context.Context, req *modelPB.DeleteModelRequest) (*modelPB.DeleteModelResponse, error) {

	eventName := "UpdateModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeleteModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	id, err := resource.GetID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeleteModelResponse{}, err
	}

	// Manually set the custom header to have a StatusNoContent http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	deleteModel, err := h.service.GetModelByID(ctx, ownerPermalink, id, modelPB.View_VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeleteModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(deleteModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.DeleteModelResponse{}, h.service.DeleteModel(ctx, ownerPermalink, id)
}

func (h *PublicHandler) RenameModel(ctx context.Context, req *modelPB.RenameModelRequest) (*modelPB.RenameModelResponse, error) {

	eventName := "RenameModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.RenameModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	id, err := resource.GetID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.RenameModelResponse{}, err
	}
	dbModel, err := h.service.RenameModel(ctx, ownerPermalink, id, req.NewModelId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.RenameModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.RenameModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.RenameModelResponse{Model: pbModel}, nil
}

func (h *PublicHandler) PublishModel(ctx context.Context, req *modelPB.PublishModelRequest) (*modelPB.PublishModelResponse, error) {

	eventName := "PublishModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.PublishModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	id, err := resource.GetID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.PublishModelResponse{}, err
	}
	dbModel, err := h.service.PublishModel(ctx, ownerPermalink, id)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.PublishModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.PublishModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.PublishModelResponse{Model: pbModel}, nil
}

func (h *PublicHandler) UnpublishModel(ctx context.Context, req *modelPB.UnpublishModelRequest) (*modelPB.UnpublishModelResponse, error) {

	eventName := "PublishModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UnpublishModelResponse{}, err
	}
	ownerPermalink := GenOwnerPermalink(owner)

	id, err := resource.GetID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UnpublishModelResponse{}, err
	}
	dbModel, err := h.service.UnpublishModel(ctx, ownerPermalink, id)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UnpublishModelResponse{}, err
	}
	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UnpublishModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, ownerPermalink)
	return &modelPB.UnpublishModelResponse{Model: pbModel}, nil
}

func (h *PublicHandler) DeployModel(ctx context.Context, req *modelPB.DeployModelRequest) (*modelPB.DeployModelResponse, error) {

	eventName := "DeployModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}

	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}

	if ownerPermalink != dbModel.Owner {
		return &modelPB.DeployModelResponse{}, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	state, err := h.service.GetResourceState(ctx, dbModel.UID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}

	if *state != modelPB.Model_STATE_OFFLINE {
		return &modelPB.DeployModelResponse{},
			status.Error(codes.FailedPrecondition, fmt.Sprintf("Deploy model only work with offline model state, current model state is %s", state))
	}

	_, err = h.service.GetTritonModels(ctx, dbModel.UID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}

	// set user desired state to STATE_ONLINE
	if _, err := h.service.UpdateModelState(ctx, dbModel.UID, &dbModel, datamodel.ModelState(modelPB.Model_STATE_ONLINE)); err != nil {
		return &modelPB.DeployModelResponse{}, err
	}

	wfId, err := h.service.DeployModelAsync(ctx, ownerPermalink, dbModel.UID)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.DeployModelResponse{}, st.Err()
	}

	if err := h.service.UpdateResourceState(
		ctx,
		dbModel.UID,
		modelPB.Model_STATE_UNSPECIFIED,
		nil,
		&wfId,
	); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.DeployModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfId),
			},
		}),
	)))

	return &modelPB.DeployModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) UndeployModel(ctx context.Context, req *modelPB.UndeployModelRequest) (*modelPB.UndeployModelResponse, error) {

	eventName := "UndeployModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	if ownerPermalink != dbModel.Owner {
		return &modelPB.UndeployModelResponse{}, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	state, err := h.service.GetResourceState(ctx, dbModel.UID)

	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	if *state != modelPB.Model_STATE_ONLINE {
		span.SetStatus(1, fmt.Sprintf("undeploy model only work with online model instance state, current model state is %s",
			state))
		return &modelPB.UndeployModelResponse{},
			status.Error(codes.FailedPrecondition, fmt.Sprintf("undeploy model only work with online model instance state, current model state is %s",
				state))
	}

	// set user desired state to STATE_OFFLINE
	if _, err := h.service.UpdateModelState(ctx, dbModel.UID, &dbModel, datamodel.ModelState(modelPB.Model_STATE_OFFLINE)); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	wfId, err := h.service.UndeployModelAsync(ctx, ownerPermalink, dbModel.UID)
	if err != nil {
		// Manually set the custom header to have a StatusUnprocessableEntity http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusUnprocessableEntity))); err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.UndeployModelResponse{}, status.Errorf(codes.Internal, err.Error())
		}
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	if err := h.service.UpdateResourceState(
		ctx,
		dbModel.UID,
		modelPB.Model_STATE_UNSPECIFIED,
		nil,
		&wfId,
	); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.UndeployModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfId),
			},
		}),
	)))

	return &modelPB.UndeployModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PublicHandler) WatchModel(ctx context.Context, req *modelPB.WatchModelRequest) (*modelPB.WatchModelResponse, error) {

	eventName := "WatchModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.WatchModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return &modelPB.WatchModelResponse{}, err
	}

	// check permission
	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return &modelPB.WatchModelResponse{}, err
	}

	state, err := h.service.GetResourceState(ctx, dbModel.UID)

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return &modelPB.WatchModelResponse{}, err
	}

	return &modelPB.WatchModelResponse{
		State: *state,
	}, err
}

func (h *PublicHandler) TestModelBinaryFileUpload(stream modelPB.ModelPublicService_TestModelBinaryFileUploadServer) error {

	eventName := "TestModelBinaryFileUpload"

	ctx, span := tracer.Start(stream.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(stream.Context(), h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	ownerPermalink := "users/" + owner.GetUid()

	triggerInput, modelID, err := savePredictInputsTestMode(stream)
	if err != nil {
		span.SetStatus(1, err.Error())
		return status.Error(codes.Internal, err.Error())
	}

	modelInDB, err := h.service.GetModelByID(stream.Context(), ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	numberOfInferences := 1
	switch modelPB.Model_Task(modelInDB.Task) {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_KEYPOINT:
		numberOfInferences = len(triggerInput.([][]byte))
	}

	// check whether model support batching or not. If not, raise an error
	if numberOfInferences > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(stream.Context(), modelInDB.UID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			span.SetStatus(1, err.Error())
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.Model_Task(modelInDB.Task)
	response, err := h.service.ModelInferTestMode(stream.Context(), ownerPermalink, modelInDB.UID, triggerInput, task)
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
		span.SetStatus(1, st.Err().Error())
		return st.Err()
	}

	err = stream.SendAndClose(&modelPB.TestModelBinaryFileUploadResponse{
		Task:        task,
		TaskOutputs: response,
	})

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(modelInDB),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return err
}

func (h *PublicHandler) TriggerModelBinaryFileUpload(stream modelPB.ModelPublicService_TriggerModelBinaryFileUploadServer) error {

	eventName := "TriggerModelBinaryFileUpload"

	ctx, span := tracer.Start(stream.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwnerWithAPIToken(stream.Context(), h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	ownerPermalink := "users/" + owner.GetUid()

	triggerInput, modelID, err := savePredictInputsTriggerMode(stream)
	if err != nil {
		span.SetStatus(1, err.Error())
		return status.Error(codes.Internal, err.Error())
	}

	modelInDB, err := h.service.GetModelByID(stream.Context(), ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// check whether model support batching or not. If not, raise an error
	numberOfInferences := 1
	switch modelPB.Model_Task(modelInDB.Task) {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_KEYPOINT:
		numberOfInferences = len(triggerInput.([][]byte))
	}
	if numberOfInferences > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(stream.Context(), modelInDB.UID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			span.SetStatus(1, err.Error())
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.Model_Task(modelInDB.Task)
	response, err := h.service.ModelInfer(stream.Context(), modelInDB.UID, triggerInput, task)
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
		span.SetStatus(1, st.Err().Error())
		return st.Err()
	}

	err = stream.SendAndClose(&modelPB.TriggerModelBinaryFileUploadResponse{
		Task:        task,
		TaskOutputs: response,
	})

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(modelInDB),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return err
}

func (h *PublicHandler) TriggerModel(ctx context.Context, req *modelPB.TriggerModelRequest) (*modelPB.TriggerModelResponse, error) {

	eventName := "TriggerModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwnerWithAPIToken(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TriggerModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TriggerModelResponse{}, err
	}

	modelInDB, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TriggerModelResponse{}, err
	}

	var inputInfer interface{}
	var lenInputs = 1
	switch modelPB.Model_Task(modelInDB.Task) {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TriggerModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(req)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TriggerModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textToImage
	case modelPB.Model_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(req)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TriggerModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(ctx, modelInDB.UID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TriggerModelResponse{}, err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TriggerModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return &modelPB.TriggerModelResponse{}, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}
	task := modelPB.Model_Task(modelInDB.Task)
	response, err := h.service.ModelInfer(ctx, modelInDB.UID, inputInfer, task)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.TriggerModelResponse{}, st.Err()
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(modelInDB),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.TriggerModelResponse{
		Task:        task,
		TaskOutputs: response,
	}, nil
}

func (h *PublicHandler) TestModel(ctx context.Context, req *modelPB.TestModelRequest) (*modelPB.TestModelResponse, error) {

	eventName := "TestModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TestModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TestModelResponse{}, err
	}

	modelInDB, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.TestModelResponse{}, err
	}

	var inputInfer interface{}
	var lenInputs = 1
	switch modelPB.Model_Task(modelInDB.Task) {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(ctx, &modelPB.TriggerModelRequest{
			Name:       req.Name,
			TaskInputs: req.TaskInputs,
		})
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TestModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(&modelPB.TriggerModelRequest{
			Name:       req.Name,
			TaskInputs: req.TaskInputs,
		})
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TestModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textToImage
	case modelPB.Model_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(
			&modelPB.TriggerModelRequest{
				Name:       req.Name,
				TaskInputs: req.TaskInputs,
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TestModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGeneration
	}

	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		tritonModelInDB, err := h.service.GetTritonEnsembleModel(ctx, modelInDB.UID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TestModelResponse{}, err
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			span.SetStatus(1, err.Error())
			return &modelPB.TestModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return &modelPB.TestModelResponse{}, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	task := modelPB.Model_Task(modelInDB.Task)
	response, err := h.service.ModelInferTestMode(ctx, ownerPermalink, modelInDB.UID, inputInfer, task)
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.TestModelResponse{}, st.Err()
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(modelInDB),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.TestModelResponse{
		Task:        task,
		TaskOutputs: response,
	}, nil
}

func inferModelByUpload(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string, mode string) {

	eventName := "InferModelByUpload"

	ctx, span := tracer.Start(req.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")

	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		span.SetStatus(1, "")
		return
	}

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	owner, err := resource.GetOwnerCustom(req, mgmtPrivateServiceClient, redisClient)
	if err != nil {
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			span.SetStatus(1, "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'jwt-sub' or 'owner-id' not found in your header")
			span.SetStatus(1, "Required parameter 'jwt-sub' or 'owner-id' not found in your header")
			return
		}
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelName := pathParams["name"]
	if modelName == "" {
		makeJSONResponse(w, 422, "Required parameter missing", "Required parameter model name not found")
		span.SetStatus(1, "Required parameter model name not found")
		return
	}

	modelID, err := resource.GetModelID(modelName)
	if err != nil {
		makeJSONResponse(w, 400, "Parameter invalid", "Required parameter instance_name is invalid")
		span.SetStatus(1, "Required parameter instance_name is invalid")
		return
	}

	modelInDB, err := s.GetModelByID(req.Context(), ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		makeJSONResponse(w, 404, "Model not found", "The model not found in server")
		span.SetStatus(1, "The model not found in server")
		return
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 400, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		return
	}

	var inputInfer interface{}
	var lenInputs = 1
	switch modelPB.Model_Task(modelInDB.Task) {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_UNSPECIFIED:
		imageInput, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			makeJSONResponse(w, 400, "File Input Error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseImageFormDataTextToImageInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		lenInputs = 1
		inputInfer = textToImage
	case modelPB.Model_TASK_TEXT_GENERATION:
		textGeneration, err := parseTextFormDataTextGenerationInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		lenInputs = 1
		inputInfer = textGeneration
	}

	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		tritonModelInDB, err := s.GetTritonEnsembleModel(req.Context(), modelInDB.UID)
		if err != nil {
			makeJSONResponse(w, 404, "Triton Model Error", fmt.Sprintf("The triton model corresponding to model %v do not exist", modelInDB.ID))
			span.SetStatus(1, fmt.Sprintf("The triton model corresponding to model %v do not exist", modelInDB.ID))
			return
		}
		configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
		doSupportBatch, err := util.DoSupportBatch(configPbFilePath)
		if err != nil {
			makeJSONResponse(w, 400, "Batching Support Error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		if !doSupportBatch {
			makeJSONResponse(w, 400, "Batching Support Error", "The model do not support batching, so could not make inference with multiple images")
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return
		}
	}

	task := modelPB.Model_Task(modelInDB.Task)
	var response []*modelPB.TaskOutput
	if mode == "test" {
		response, err = s.ModelInferTestMode(req.Context(), ownerPermalink, modelInDB.UID, inputInfer, task)
	} else {
		response, err = s.ModelInfer(req.Context(), modelInDB.UID, inputInfer, task)
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
		span.SetStatus(1, st.Message())
		return
	}

	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(200)
	res, err := util.MarshalOptions.Marshal(&modelPB.TestModelBinaryFileUploadResponse{
		Task:        task,
		TaskOutputs: response,
	})
	if err != nil {
		makeJSONResponse(w, 500, "Error Predict Model", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(modelInDB),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	_, _ = w.Write(res)

}

func HandleTestModelByUpload(s service.Service, w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelByUpload(s, w, r, pathParams, "test")
}

func HandleTriggerModelByUpload(s service.Service, w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelByUpload(s, w, r, pathParams, "trigger")
}

func (h *PublicHandler) GetModelCard(ctx context.Context, req *modelPB.GetModelCardRequest) (*modelPB.GetModelCardResponse, error) {

	eventName := "GetModelCard"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelCardResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelCardResponse{}, err
	}

	dbModel, err := h.service.GetModelByID(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelCardResponse{}, err
	}

	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md", config.Config.TritonServer.ModelStore, owner, modelID)
	stat, err := os.Stat(readmeFilePath)
	if err != nil { // return empty content base64
		span.SetStatus(1, err.Error())
		return &modelPB.GetModelCardResponse{
			Readme: &modelPB.ModelCard{
				Name:     req.Name,
				Size:     0,
				Type:     "file",
				Encoding: "base64",
				Content:  []byte(""),
			},
		}, nil
	}

	content, _ := os.ReadFile(readmeFilePath)

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.GetModelCardResponse{Readme: &modelPB.ModelCard{
		Name:     req.Name,
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Content:  []byte(content),
	}}, nil
}

func (h *PublicHandler) GetModelDefinition(ctx context.Context, req *modelPB.GetModelDefinitionRequest) (*modelPB.GetModelDefinitionResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	definitionID, err := resource.GetDefinitionID(req.Name)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	dbModelDefinition, err := h.service.GetModelDefinition(ctx, definitionID)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	pbModelDefinition := DBModelDefinitionToPBModelDefinition(ctx, &dbModelDefinition)
	return &modelPB.GetModelDefinitionResponse{ModelDefinition: pbModelDefinition}, nil
}

func (h *PublicHandler) ListModelDefinitions(ctx context.Context, req *modelPB.ListModelDefinitionsRequest) (*modelPB.ListModelDefinitionsResponse, error) {

	ctx, span := tracer.Start(ctx, "ListModelDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	dbModelDefinitions, nextPageToken, totalSize, err := h.service.ListModelDefinitions(ctx, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelDefinitionsResponse{}, err
	}

	pbDefinitions := []*modelPB.ModelDefinition{}
	for _, dbModelDefinition := range dbModelDefinitions {
		pbDefinitions = append(pbDefinitions, DBModelDefinitionToPBModelDefinition(ctx, &dbModelDefinition))
	}

	resp := modelPB.ListModelDefinitionsResponse{
		ModelDefinitions: pbDefinitions,
		NextPageToken:    nextPageToken,
		TotalSize:        totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) GetModelOperation(ctx context.Context, req *modelPB.GetModelOperationRequest) (*modelPB.GetModelOperationResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelOperation",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	operationId, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationId)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}

	return &modelPB.GetModelOperationResponse{
		Operation: operation,
	}, nil
}
