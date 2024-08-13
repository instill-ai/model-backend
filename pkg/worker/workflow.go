package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/utils"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"

	"github.com/instill-ai/x/errmsg"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type InferInput any

type TriggerModelWorkflowRequest struct {
	TriggerUID         uuid.UUID
	ModelID            string
	ModelUID           uuid.UUID
	ModelVersion       datamodel.ModelVersion
	OwnerUID           uuid.UUID
	OwnerType          string
	UserUID            uuid.UUID
	UserType           string
	ModelDefinitionUID uuid.UUID
	RequesterUID       uuid.UUID
	Task               commonpb.Task
	ParsedInputKey     string
	Mode               mgmtpb.Mode
	Hardware           string
	Visibility         datamodel.ModelVisibility
	InputReferenceID   string
	Source             datamodel.TriggerSource
}

type TriggerModelActivityRequest struct {
	TriggerModelWorkflowRequest
	WorkflowExecutionID string
}

type TriggerModelWorkflowResponse struct {
	TriggerModelActivityResponse
}

type TriggerModelActivityResponse struct {
	TaskOutputBytes []byte
	OutputKey       string
}

var tracer = otel.Tracer("model-backend.temporal.tracer")

func (w *worker) TriggerModelWorkflow(ctx workflow.Context, param *TriggerModelWorkflowRequest) (*TriggerModelWorkflowResponse, error) {

	startTime := time.Now()
	eventName := "TriggerModelWorkflow"

	sCtx, span := tracer.Start(context.Background(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := custom_logger.GetZapLogger(sCtx)
	logger.Info("TriggerModelWorkflow started")

	var ownerType mgmtpb.OwnerType
	switch param.OwnerType {
	case "organizations":
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION
	case "users":
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	var usageData *utils.UsageMetricData
	if param.Mode == mgmtpb.Mode_MODE_ASYNC {
		usageData = &utils.UsageMetricData{
			TriggerUID:         param.TriggerUID.String(),
			OwnerUID:           param.OwnerUID.String(),
			OwnerType:          ownerType,
			UserUID:            param.UserUID.String(),
			UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
			ModelUID:           param.ModelUID.String(),
			Version:            param.ModelVersion.Version,
			Mode:               param.Mode,
			ModelDefinitionUID: param.ModelDefinitionUID.String(),
			ModelTask:          param.Task,
			TriggerTime:        startTime.Format(time.RFC3339Nano),
		}
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var triggerResult TriggerModelActivityResponse
	if err := workflow.ExecuteActivity(ctx, w.TriggerModelActivity, &TriggerModelActivityRequest{
		TriggerModelWorkflowRequest: *param,
		WorkflowExecutionID:         workflow.GetInfo(ctx).WorkflowExecution.ID,
	}).Get(ctx, &triggerResult); err != nil {
		if param.Mode == mgmtpb.Mode_MODE_ASYNC {
			w.writeErrorDataPoint(sCtx, err, span, startTime, usageData)
		}
		logger.Error(w.toApplicationError(err, param.ModelID, ModelWorkflowError).Error())
		return nil, w.toApplicationError(err, param.ModelID, ModelWorkflowError)
	}

	if param.Mode == mgmtpb.Mode_MODE_ASYNC {
		usageData.ComputeTimeDuration = time.Since(startTime).Seconds()
		usageData.Status = mgmtpb.Status_STATUS_COMPLETED
		if err := w.writeNewDataPoint(sCtx, usageData); err != nil {
			logger.Warn(err.Error())
		}
	}

	logger.Info("TriggerModelWorkflow completed")

	return &TriggerModelWorkflowResponse{
		TriggerModelActivityResponse: triggerResult,
	}, nil
}

func (w *worker) TriggerModelActivity(ctx context.Context, param *TriggerModelActivityRequest) (*TriggerModelActivityResponse, error) {

	eventName := "TriggerModelActivity"

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})
	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := custom_logger.GetZapLogger(ctx)
	logger.Info("TriggerModelActivity started")

	// TODO: temporary disable usage check until further decision
	// if err := w.modelUsageHandler.Check(ctx, &usage.ModelUsageHandlerParams{
	// 	UserUID:      param.UserUID,
	// 	OwnerUID:     param.OwnerUID,
	// 	RequesterUID: param.RequesterUID,
	// }); err != nil {
	// 	return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	// }

	start := time.Now()

	runLog, err := w.repository.CreateModelTrigger(ctx, &datamodel.ModelTrigger{
		ModelUID:         param.ModelUID,
		ModelVersion:     param.ModelVersion.Version,
		Status:           datamodel.TriggerStatus(modelpb.ModelRun_RUN_STATUS_PROCESSING),
		Source:           param.Source,
		RequesterUID:     param.RequesterUID,
		InputReferenceID: param.InputReferenceID,
	})
	if err != nil {
		logger.Error("CreateModelTrigger in DB failed", zap.String("TriggerUID", param.TriggerUID.String()), zap.Error(err))
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	succeeded := false
	defer func() {
		if err != nil || !succeeded {
			runLog.Status = datamodel.TriggerStatus(modelpb.ModelRun_RUN_STATUS_FAILED)
			endTime := time.Now()
			timeUsed := endTime.Sub(start)
			runLog.TotalDuration = null.IntFrom(timeUsed.Milliseconds())
			runLog.EndTime = null.TimeFrom(endTime)
			if err != nil {
				runLog.Error = null.StringFrom(err.Error())
			} else {
				runLog.Error = null.StringFrom("unknown error occurred")
			}
			if err = w.repository.UpdateModelTrigger(ctx, runLog); err != nil {
				logger.Error("UpdateModelTrigger for TriggerModelActivity failed", zap.Error(err))
			}
		}
	}()

	modelName := fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID)

	modelMetadataResponse := w.ray.ModelMetadataRequest(ctx, modelName, param.ModelVersion.Version)
	if modelMetadataResponse == nil {
		err = fmt.Errorf("model is offline") // used by model run logging in defer
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	blob, err := w.redisClient.Get(ctx, param.ParsedInputKey).Bytes()
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}
	defer func() {
		w.redisClient.Del(ctx, param.ParsedInputKey)
		w.redisClient.ExpireGT(
			ctx,
			fmt.Sprintf("model_trigger_input:%s:%s", param.UserUID, param.ModelUID.String()),
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		w.redisClient.ExpireGT(
			ctx,
			fmt.Sprintf("model_trigger_output_key:%s:%s", param.UserUID, param.ModelUID.String()),
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}()

	var inferInput InferInput
	switch param.Task {
	case commonpb.Task_TASK_CLASSIFICATION,
		commonpb.Task_TASK_DETECTION,
		commonpb.Task_TASK_INSTANCE_SEGMENTATION,
		commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
		commonpb.Task_TASK_OCR,
		commonpb.Task_TASK_KEYPOINT,
		commonpb.Task_TASK_UNSPECIFIED:
		var input [][]byte
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		var input *ray.TextToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		var input *ray.ImageToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		var input *ray.VisualQuestionAnsweringInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		var input *ray.TextGenerationChatInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_GENERATION:
		var input *ray.TextGenerationInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	}

	logger.Info("ModelInferRequest started", zap.String("modelName", modelName), zap.String("modelVersion", param.ModelVersion.Version))

	inferResponse, err := w.ray.ModelInferRequest(ctx, param.Task, inferInput, modelName, param.ModelVersion.Version, modelMetadataResponse)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	endTime := time.Now()
	timeUsed := endTime.Sub(start)
	logger.Info("ModelInferRequest ended", zap.Duration("timeUsed", timeUsed))

	// TODO: temporary disable usage collect until further decision
	// if err = w.modelUsageHandler.Collect(ctx, &usage.ModelUsageHandlerParams{
	// 	UserUID:        param.UserUID,
	// 	OwnerUID:       param.OwnerUID,
	// 	ModelUID:       param.ModelUID,
	// 	ModelVersion:   param.ModelVersion.Version,
	// 	ModelTriggerID: param.TriggerUID.String(),
	// 	ModelID:        param.ModelID,
	// 	UsageTime:      timeUsed,
	// 	Hardware:       param.Hardware,
	// 	RequesterUID:   param.RequesterUID,
	// }); err != nil {
	// 	return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	// }

	outputs, err := ray.PostProcess(inferResponse, modelMetadataResponse, param.Task)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	triggerModelResp := &modelpb.TriggerUserModelResponse{
		Task:        param.Task,
		TaskOutputs: outputs,
	}

	outputJSON, err := protojson.Marshal(triggerModelResp)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}
	logger.Debug("test by jeremy", zap.String("outputJSON", string(outputJSON)))

	outputKey := fmt.Sprintf("async_model_response:%s", param.WorkflowExecutionID)
	w.redisClient.Set(
		ctx,
		outputKey,
		outputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	jsonOutput, err := json.Marshal(outputs)
	if err != nil {
		logger.Warn("json marshal error for task inputs")
	}

	outputReferenceUID, _ := uuid.NewV4()
	outputReferenceID := outputReferenceUID.String()
	// todo: store url and file size
	_, _, err = w.minioClient.UploadFileBytes(ctx, outputReferenceID, outputJSON, constant.ContentTypeJSON)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	runLog.TotalDuration = null.IntFrom(timeUsed.Milliseconds())
	runLog.EndTime = null.TimeFrom(endTime)
	runLog.OutputReferenceID = null.StringFrom(outputReferenceID)
	runLog.Status = datamodel.TriggerStatus(modelpb.ModelRun_RUN_STATUS_COMPLETED)
	if err = w.repository.UpdateModelTrigger(ctx, runLog); err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	succeeded = true
	logger.Info("TriggerModelActivity completed")

	return &TriggerModelActivityResponse{
		// todo: this is not used anymore?
		TaskOutputBytes: jsonOutput,
		OutputKey:       outputKey,
	}, nil
}

func (w *worker) writeErrorDataPoint(ctx context.Context, err error, span trace.Span, startTime time.Time, dataPoint *utils.UsageMetricData) {
	span.SetStatus(1, err.Error())
	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_ERRORED
	_ = w.writeNewDataPoint(ctx, dataPoint)
}

// func (w *worker) writeErrorPrediction(ctx context.Context, err error, span trace.Span, startTime time.Time, pred *datamodel.ModelPrediction) {
// 	span.SetStatus(1, err.Error())
// 	pred.ComputeTimeDuration = time.Since(startTime).Seconds()
// 	pred.Status = datamodel.Status(mgmtpb.Status_STATUS_ERRORED)
// 	_ = w.writePrediction(ctx, pred)
// }

// toApplicationError wraps a temporal task error in a temporal.Application
// error, adding end-user information that can be extracted by the temporal
// client.
func (w *worker) toApplicationError(err error, modelID, errType string) error {
	details := EndUserErrorDetails{
		// If no end-user message is present in the error, MessageOrErr will
		// return the string version of the error. For an end user, this extra
		// information is more actionable than no information at all.
		Message: fmt.Sprintf("Model %s failed to execute. %s", modelID, errmsg.MessageOrErr(err)),
	}
	return temporal.NewApplicationErrorWithCause("model failed to execute", errType, err, details)
}

const (
	ModelWorkflowError = "ModelWorkflowError"
	ModelActivityError = "ModelActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}

type UploadToMinioActivityParam struct {
	ObjectName  string
	Data        []byte
	ContentType string
}

func (w *worker) UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityParam) (string, error) {

	// url, _, err := w.repository.UploadToMinio(ctx, param.ObjectName, param.Data, param.ContentType, param.BucketName)
	// return url, err
	return "", nil
}
