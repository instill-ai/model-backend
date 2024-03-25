package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func savePredictInputsTriggerMode(stream modelPB.ModelPublicService_TriggerUserModelBinaryFileUploadServer) (triggerInput any, modelID string, version string, err error) {

	var firstChunk = true

	var fileData *modelPB.TriggerUserModelBinaryFileUploadRequest

	var allContentFiles []byte
	var fileLengths []uint32

	var textToImageInput *ray.TextToImageInput
	var textGeneration *ray.TextGenerationInput

	var task *modelPB.TaskInputStream
	for {
		fileData, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return nil, "", "", err
		}

		if firstChunk { // first chunk contains model instance name
			firstChunk = false
			modelID, err = resource.GetRscNameID(fileData.Name) // format "users/{user}/models/{model}"
			if err != nil {
				return nil, "", "", err
			}
			version = fileData.Version
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
				extraParams := ""
				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
					if err != nil {
						log.Fatalf("Error marshaling to JSON: %v", err)
					} else {
						extraParams = string(jsonData)
					}
				}
				textToImageInput = &ray.TextToImageInput{
					Prompt:      fileData.TaskInput.GetTextToImage().Prompt,
					PromptImage: "", // TODO: support streaming image generation
					Steps:       *fileData.TaskInput.GetTextToImage().Steps,
					CfgScale:    *fileData.TaskInput.GetTextToImage().CfgScale,
					Seed:        *fileData.TaskInput.GetTextToImage().Seed,
					Samples:     *fileData.TaskInput.GetTextToImage().Samples,
					ExtraParams: extraParams, // *fileData.TaskInput.GetTextToImage().ExtraParams
				}
			case *modelPB.TaskInputStream_TextGeneration:
				extraParams := ""
				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
					if err != nil {
						log.Fatalf("Error marshaling to JSON: %v", err)
					} else {
						extraParams = string(jsonData)
					}
				}
				textGeneration = &ray.TextGenerationInput{
					Prompt: fileData.TaskInput.GetTextGeneration().Prompt,
					// PromptImage:  "", // TODO: support streaming image generation
					MaxNewTokens: *fileData.TaskInput.GetTextGeneration().MaxNewTokens,
					// StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
					Temperature: *fileData.TaskInput.GetTextGeneration().Temperature,
					TopK:        *fileData.TaskInput.GetTextGeneration().TopK,
					Seed:        *fileData.TaskInput.GetTextGeneration().Seed,
					ExtraParams: extraParams, // *fileData.TaskInput.GetTextGeneration().ExtraParams,
				}
			default:
				return nil, "", "", errors.New("unsupported task input type")
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
				return nil, "", "", errors.New("unsupported task input type")
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
			return nil, "", "", errors.New("wrong parameter length of files")
		}
		imageBytes := make([][]byte, len(fileLengths))
		start := uint32(0)
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
		return imageBytes, modelID, "", nil
	case *modelPB.TaskInputStream_TextToImage:
		return textToImageInput, modelID, "", nil
	case *modelPB.TaskInputStream_TextGeneration:
		return textGeneration, modelID, "", nil
	}
	return nil, "", "", errors.New("unsupported task input type")
}

func (h *PublicHandler) TriggerUserModelBinaryFileUpload(stream modelPB.ModelPublicService_TriggerUserModelBinaryFileUploadServer) error {

	startTime := time.Now()
	eventName := "TriggerUserModelBinaryFileUpload"

	ctx, span := tracer.Start(stream.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	triggerInput, path, version, err := savePredictInputsTriggerMode(stream)
	if err != nil {
		span.SetStatus(1, err.Error())
		return status.Error(codes.Internal, err.Error())
	}

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(path)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	pbModel, err := h.service.GetNamespaceModelByID(stream.Context(), ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return status.Error(codes.InvalidArgument, err.Error())
	}

	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtPB.OwnerType_OWNER_TYPE_USER,
		UserUID:            authUser.UID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	// check whether model support batching or not. If not, raise an error
	numberOfInferences := 1
	switch pbModel.Task {
	case commonPB.Task_TASK_CLASSIFICATION,
		commonPB.Task_TASK_DETECTION,
		commonPB.Task_TASK_INSTANCE_SEGMENTATION,
		commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
		commonPB.Task_TASK_OCR,
		commonPB.Task_TASK_KEYPOINT:
		numberOfInferences = len(triggerInput.([][]byte))
	}
	if numberOfInferences > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			return status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	response, err := h.service.TriggerNamespaceModelByID(stream.Context(), ns, authUser, modelID, version, triggerInput, pbModel.Task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
			"Ray inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Ray inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		usageData.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewDataPoint(ctx, usageData)
		return st.Err()
	}

	err = stream.SendAndClose(&modelPB.TriggerUserModelBinaryFileUploadResponse{
		Task:        pbModel.Task,
		TaskOutputs: response,
	})

	usageData.Status = mgmtPB.Status_STATUS_COMPLETED
	if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
		logger.Warn("usage and metric data write fail")
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return err
}
