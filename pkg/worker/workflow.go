package worker

import (
	"context"
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

	miniogo "github.com/minio/minio-go/v7"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/usage"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	miniox "github.com/instill-ai/x/minio"
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
	Mode               mgmtpb.Mode
	Hardware           string
	Visibility         datamodel.ModelVisibility
	RunLog             *datamodel.ModelRun
	ExpiryRuleTag      string
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
		if param.Mode == mgmtpb.Mode_MODE_ASYNC {
			w.redisClient.ExpireGT(
				sCtx,
				fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", param.UserUID, param.RequesterUID, param.ModelUID.String(), param.ModelVersion.Version),
				time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
			)
			w.redisClient.ExpireGT(
				sCtx,
				fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", param.UserUID, param.RequesterUID, param.ModelUID.String(), ""),
				time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
			)
		}
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

	var err error

	// param.RunLog.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING)

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})
	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := custom_logger.GetZapLogger(ctx)
	logger.Info("TriggerModelActivity started")

	if err = w.modelUsageHandler.Check(ctx, &usage.ModelUsageHandlerParams{
		UserUID:      param.UserUID,
		OwnerUID:     param.OwnerUID,
		RequesterUID: param.RequesterUID,
	}); err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	// wait for model instance to come online to start processing the request
	// temporary solution to not overcharge for credits
	// TODO: design a better flow
	state, _, numOfActiveReplica, err := w.ray.ModelReady(ctx, fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID, param.ModelID), param.ModelVersion.Version)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}
	for *state == modelpb.State_STATE_OFFLINE {
		time.Sleep(time.Millisecond * 500)
		state, _, numOfActiveReplica, err = w.ray.ModelReady(ctx, fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID, param.ModelID), param.ModelVersion.Version)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
	}
	for *state != modelpb.State_STATE_ACTIVE || numOfActiveReplica <= 0 {
		logger.Debug(fmt.Sprintf("model upscale state: %v", state))
		logger.Debug(fmt.Sprintf("model upscale numOfActiveReplica: %v", numOfActiveReplica))
		if state, _, numOfActiveReplica, err = w.ray.ModelReady(ctx, fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID, param.ModelID), param.ModelVersion.Version); err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		} else if *state != modelpb.State_STATE_SCALING_UP && *state != modelpb.State_STATE_STARTING && *state != modelpb.State_STATE_ACTIVE {
			logger.Error(fmt.Sprintf("model upscale failed: current model state: %v", state), zap.Error(err))
			return w.toApplicationError(fmt.Errorf("model upscale failed: current model state: %v", state), param.ModelID, ModelActivityError)
		} else {
			time.Sleep(time.Millisecond * 500)
		}
	}

	start := time.Now()

	succeeded := false
	defer func() {
		if err != nil || !succeeded {
			param.RunLog.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED)
			endTime := time.Now()
			timeUsed := endTime.Sub(start)
			param.RunLog.TotalDuration = null.IntFrom(timeUsed.Milliseconds())
			param.RunLog.EndTime = null.TimeFrom(endTime)
			if err != nil {
				param.RunLog.Error = null.StringFrom(err.Error())
			} else {
				param.RunLog.Error = null.StringFrom("unknown error occurred")
			}
			if err = w.repository.UpdateModelRun(ctx, param.RunLog); err != nil {
				logger.Error("UpdateModelRun for TriggerModelActivity failed", zap.Error(err))
			}
		}
	}()

	modelName := fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID)

	input, err := w.minioClient.GetFile(ctx, nil, param.RunLog.InputReferenceID)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	triggerModelReq := &modelpb.TriggerNamespaceModelRequest{}
	if err := protojson.Unmarshal(input, triggerModelReq); err != nil {
		return err
	}

	logger.Info("ModelInferRequest started", zap.String("modelName", modelName), zap.String("modelVersion", param.ModelVersion.Version))

	inferResponse, err := w.ray.ModelInferRequest(ctx, param.Task, triggerModelReq, modelName, param.ModelVersion.Version)
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	for _, o := range inferResponse.GetTaskOutputs() {
		err := datamodel.ValidateJSONSchema(datamodel.TasksJSONOutputSchemaMap[param.Task.String()], o, false)
		if err != nil {
			return w.toApplicationError(err, param.ModelID, ModelActivityError)
		}
	}

	triggerModelResp := &modelpb.TriggerNamespaceModelResponse{
		Task:        param.Task,
		TaskOutputs: inferResponse.GetTaskOutputs(),
	}

	outputJSON, err := protojson.Marshal(triggerModelResp)
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
		ModelRunUID:    param.RunLog.UID,
		ModelVersion:   param.ModelVersion.Version,
		ModelTriggerID: param.TriggerUID.String(),
		ModelID:        param.ModelID,
		UsageTime:      timeUsed,
		Hardware:       param.Hardware,
		RequesterUID:   param.RequesterUID,
	}); err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	outputReferenceID := miniox.GenerateOutputRefID("model-runs")
	// todo: put it in separate workflow activity and store url and file size
	_, _, err = w.minioClient.UploadFileBytes(ctx, logger, &miniox.UploadFileBytesParam{
		FilePath:      outputReferenceID,
		FileBytes:     outputJSON,
		FileMimeType:  constant.ContentTypeJSON,
		ExpiryRuleTag: param.ExpiryRuleTag,
	})
	if err != nil {
		return w.toApplicationError(err, param.ModelID, ModelActivityError)
	}

	param.RunLog.TotalDuration = null.IntFrom(timeUsed.Milliseconds())
	param.RunLog.EndTime = null.TimeFrom(endTime)
	param.RunLog.OutputReferenceID = null.StringFrom(outputReferenceID)
	param.RunLog.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED)
	if err = w.repository.UpdateModelRun(ctx, param.RunLog); err != nil {
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
	ObjectName    string
	Data          []byte
	ContentType   string
	ExpiryRuleTag string
}

type UploadToMinioActivityResponse struct {
	URL        string
	ObjectInfo *miniogo.ObjectInfo
}

func (w *worker) UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityRequest) (*UploadToMinioActivityResponse, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)
	logger.Info("UploadToMinioActivity started")

	url, objectInfo, err := w.minioClient.UploadFileBytes(ctx, logger, &miniox.UploadFileBytesParam{
		FilePath:      param.ObjectName,
		FileBytes:     param.Data,
		FileMimeType:  param.ContentType,
		ExpiryRuleTag: param.ExpiryRuleTag,
	})
	if err != nil {
		return nil, err
	}
	return &UploadToMinioActivityResponse{
		URL:        url,
		ObjectInfo: objectInfo,
	}, nil
}
