package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"
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
	"github.com/instill-ai/model-backend/pkg/usage"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	minio2 "github.com/instill-ai/model-backend/pkg/minio"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
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

var tracer = otel.Tracer("model-backend.temporal.tracer")

func (w *worker) TriggerModelWorkflow(ctx workflow.Context, param *TriggerModelWorkflowRequest) error {

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

	defer func() {
		w.redisClient.Del(sCtx, param.ParsedInputKey)
		w.redisClient.ExpireGT(
			sCtx,
			fmt.Sprintf("model_trigger_output_key:%s:%s:%s", param.UserUID, param.ModelUID.String(), param.ModelVersion.Version),
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		w.redisClient.ExpireGT(
			sCtx,
			fmt.Sprintf("model_trigger_output_key:%s:%s:%s", param.UserUID, param.ModelUID.String(), ""),
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}()

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if err := workflow.ExecuteActivity(ctx, w.TriggerModelActivity, &TriggerModelActivityRequest{
		TriggerModelWorkflowRequest: *param,
		WorkflowExecutionID:         workflow.GetInfo(ctx).WorkflowExecution.ID,
	}).Get(ctx, nil); err != nil {
		if param.Mode == mgmtpb.Mode_MODE_ASYNC {
			w.writeErrorDataPoint(sCtx, err, span, startTime, usageData)
		}
		_ = workflow.UpsertMemo(ctx, map[string]any{
			"error": fmt.Sprintf("Model %s failed to execute. %s", param.ModelID, errmsg.MessageOrErr(err)),
		})

		logger.Error(w.toApplicationError(err, param.ModelID, ModelWorkflowError).Error())
		return w.toApplicationError(err, param.ModelID, ModelWorkflowError)
	}

	if param.Mode == mgmtpb.Mode_MODE_ASYNC {
		usageData.ComputeTimeDuration = time.Since(startTime).Seconds()
		usageData.Status = mgmtpb.Status_STATUS_COMPLETED
		if err := w.writeNewDataPoint(sCtx, usageData); err != nil {
			logger.Warn(err.Error())
		}
	}

	logger.Info("TriggerModelWorkflow completed")

	return nil
}

func (w *worker) TriggerModelActivity(ctx context.Context, param *TriggerModelActivityRequest) error {

	eventName := "TriggerModelActivity"

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})
	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := custom_logger.GetZapLogger(ctx)
	logger.Info("TriggerModelActivity started")

	if err := w.modelUsageHandler.Check(ctx, &usage.ModelUsageHandlerParams{
		UserUID:      param.UserUID,
		OwnerUID:     param.OwnerUID,
		RequesterUID: param.RequesterUID,
	}); err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	start := time.Now()

	runLog, err := w.repository.CreateModelTrigger(ctx, &datamodel.ModelTrigger{
		BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: param.TriggerUID},
		ModelUID:             param.ModelUID,
		ModelVersion:         param.ModelVersion.Version,
		Status:               datamodel.TriggerStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:               param.Source,
		RequesterUID:         param.RequesterUID,
		InputReferenceID:     param.InputReferenceID,
	})
	if err != nil {
		logger.Error("CreateModelTrigger in DB failed", zap.String("TriggerUID", param.TriggerUID.String()), zap.Error(err))
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	succeeded := false
	defer func() {
		if err != nil || !succeeded {
			runLog.Status = datamodel.TriggerStatus(runpb.RunStatus_RUN_STATUS_FAILED)
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
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	blob, err := w.redisClient.Get(ctx, param.ParsedInputKey).Bytes()
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

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
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		var input *ray.TextToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		var input *ray.ImageToImageInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		var input *ray.VisualQuestionAnsweringInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		var input *ray.TextGenerationChatInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	case commonpb.Task_TASK_TEXT_GENERATION:
		var input *ray.TextGenerationInput
		err = json.Unmarshal(blob, &input)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
		inferInput = input
	}

	logger.Info("ModelInferRequest started", zap.String("modelName", modelName), zap.String("modelVersion", param.ModelVersion.Version))

	inferResponse, err := w.ray.ModelInferRequest(ctx, param.Task, inferInput, modelName, param.ModelVersion.Version, modelMetadataResponse)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	endTime := time.Now()
	timeUsed := endTime.Sub(start)
	logger.Info("ModelInferRequest ended", zap.Duration("timeUsed", timeUsed))

	if err = w.modelUsageHandler.Collect(ctx, &usage.ModelUsageHandlerParams{
		UserUID:        param.UserUID,
		OwnerUID:       param.OwnerUID,
		ModelUID:       param.ModelUID,
		ModelRunUID:    runLog.UID,
		ModelVersion:   param.ModelVersion.Version,
		ModelTriggerID: param.TriggerUID.String(),
		ModelID:        param.ModelID,
		UsageTime:      timeUsed,
		Hardware:       param.Hardware,
		RequesterUID:   param.RequesterUID,
	}); err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	outputs, err := ray.PostProcess(inferResponse, modelMetadataResponse, param.Task)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	triggerModelResp := &modelpb.TriggerUserModelResponse{
		Task:        param.Task,
		TaskOutputs: outputs,
	}

	outputJSON, err := protojson.Marshal(triggerModelResp)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	outputReferenceID := minio2.GenerateOutputRefID()
	// todo: put it in separate workflow activity and store url and file size
	_, _, err = w.minioClient.UploadFileBytes(ctx, outputReferenceID, outputJSON, constant.ContentTypeJSON)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	runLog.TotalDuration = null.IntFrom(timeUsed.Milliseconds())
	runLog.EndTime = null.TimeFrom(endTime)
	runLog.OutputReferenceID = null.StringFrom(outputReferenceID)
	runLog.Status = datamodel.TriggerStatus(runpb.RunStatus_RUN_STATUS_COMPLETED)
	if err = w.repository.UpdateModelTrigger(ctx, runLog); err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	succeeded = true
	logger.Info("TriggerModelActivity completed")

	return nil
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

type UploadToMinioActivityRequest struct {
	ObjectName  string
	Data        []byte
	ContentType string
}

type UploadToMinioActivityResponse struct {
	URL        string
	ObjectInfo *minio.ObjectInfo
}

func (w *worker) UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityRequest) (*UploadToMinioActivityResponse, error) {
	url, objectInfo, err := w.minioClient.UploadFileBytes(ctx, param.ObjectName, param.Data, param.ContentType)
	if err != nil {
		return nil, err
	}
	return &UploadToMinioActivityResponse{
		URL:        url,
		ObjectInfo: objectInfo,
	}, nil
}
