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
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type InferInput any

type TriggerModelWorkflowRequest struct {
	ModelID                  string
	ModelUID                 uuid.UUID
	ModelVersion             string
	OwnerUID                 uuid.UUID
	OwnerType                string
	UserUID                  uuid.UUID
	UserType                 string
	ModelDefinitionUID       uuid.UUID
	Task                     commonPB.Task
	TriggerInputBlobRedisKey string
	Mode                     mgmtPB.Mode
}

type TriggerModelActivityRequest struct {
	TriggerModelWorkflowRequest
	WorkflowExecutionID string
}

type TriggerModelWorkflowResponse struct {
	TriggerModelActivityResponse
}

type TriggerModelActivityResponse struct {
	OutputBlobRedisKey string
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

	var ownerType mgmtPB.OwnerType
	switch param.OwnerType {
	case "organizations":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION
	case "users":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.UsageMetricData{
		OwnerUID:           param.OwnerUID.String(),
		OwnerType:          ownerType,
		UserUID:            param.UserUID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER,
		Mode:               param.Mode,
		ModelDefinitionUID: param.ModelDefinitionUID.String(),
		ModelTask:          param.Task,
		TriggerTime:        startTime.Format(time.RFC3339Nano),
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Minute,
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
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, w.toApplicationError(err, param.ModelID, ModelWorkflowError)
	}

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	if err := w.writeNewDataPoint(sCtx, &dataPoint); err != nil {
		logger.Warn(err.Error())
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

	modelName := fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID)

	modelMetadataResponse := w.ray.ModelMetadataRequest(ctx, modelName, param.ModelVersion)
	if modelMetadataResponse == nil {
		return nil, w.toApplicationError(fmt.Errorf("model is offline"), param.ModelID, ModelActivityError)
	}

	blob, err := w.redisClient.Get(ctx, param.TriggerInputBlobRedisKey).Bytes()
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	var inferInput InferInput
	switch param.Task {
	case commonPB.Task_TASK_CLASSIFICATION,
		commonPB.Task_TASK_DETECTION,
		commonPB.Task_TASK_INSTANCE_SEGMENTATION,
		commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
		commonPB.Task_TASK_OCR,
		commonPB.Task_TASK_KEYPOINT,
		commonPB.Task_TASK_UNSPECIFIED:
		var input [][]byte
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		var input *ray.TextToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		var input *ray.ImageToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		var input *ray.VisualQuestionAnsweringInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:
		var input *ray.TextGenerationChatInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonPB.Task_TASK_TEXT_GENERATION:
		var input *ray.TextGenerationInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	}

	inferResponse, err := w.ray.ModelInferRequest(ctx, param.Task, inferInput, modelName, param.ModelVersion, modelMetadataResponse)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	outputs, err := ray.PostProcess(inferResponse, modelMetadataResponse, param.Task)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	triggerModelResp := &modelPB.TriggerUserModelResponse{
		Task:        param.Task,
		TaskOutputs: outputs,
	}

	outputJSON, err := protojson.Marshal(triggerModelResp)
	if err != nil {
		return nil, w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	blobRedisKey := fmt.Sprintf("async_model_response:%s", param.WorkflowExecutionID)
	w.redisClient.Set(
		ctx,
		blobRedisKey,
		outputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	logger.Info("TriggerModelActivity completed")

	w.redisClient.Del(ctx, param.TriggerInputBlobRedisKey)

	return &TriggerModelActivityResponse{
		OutputBlobRedisKey: blobRedisKey,
	}, nil
}

func (w *worker) writeErrorDataPoint(ctx context.Context, err error, span trace.Span, startTime time.Time, dataPoint *utils.UsageMetricData) {
	span.SetStatus(1, err.Error())
	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
	_ = w.writeNewDataPoint(ctx, dataPoint)
}

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
