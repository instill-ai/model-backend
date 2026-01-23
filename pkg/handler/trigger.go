package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	logx "github.com/instill-ai/x/log"
	resourcex "github.com/instill-ai/x/resource"
)

// triggerModelParams contains parsed parameters for model trigger.
type triggerModelParams struct {
	namespaceID string
	modelID     string
	version     string
	taskInputs  []*structpb.Struct
}

// parseModelVersionName parses a model version name and extracts namespace, model, and version.
// Format: namespaces/{namespace}/models/{model}/versions/{version}
func parseModelVersionName(name string) (namespaceID, modelID, version string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 6 || parts[0] != "namespaces" || parts[2] != "models" || parts[4] != "versions" {
		return "", "", "", status.Errorf(codes.InvalidArgument, "invalid model version name format: %s", name)
	}
	return parts[1], parts[3], parts[5], nil
}

// parseModelName parses a model name and extracts namespace and model.
// Format: namespaces/{namespace}/models/{model}
func parseModelName(name string) (namespaceID, modelID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "namespaces" || parts[2] != "models" {
		return "", "", status.Errorf(codes.InvalidArgument, "invalid model name format: %s", name)
	}
	return parts[1], parts[3], nil
}

// TriggerModelVersion triggers a model for a given namespace.
func (h *PublicHandler) TriggerModelVersion(ctx context.Context, req *modelpb.TriggerModelVersionRequest) (resp *modelpb.TriggerModelVersionResponse, err error) {
	resp = &modelpb.TriggerModelVersionResponse{}

	namespaceID, modelID, version, err := parseModelVersionName(req.GetName())
	if err != nil {
		return nil, err
	}

	params := triggerModelParams{
		namespaceID: namespaceID,
		modelID:     modelID,
		version:     version,
		taskInputs:  req.GetTaskInputs(),
	}

	resp.Task, resp.TaskOutputs, err = h.triggerModel(ctx, params)

	return resp, err
}

// TriggerModel triggers a model for a given namespace.
func (h *PublicHandler) TriggerModel(ctx context.Context, req *modelpb.TriggerModelRequest) (resp *modelpb.TriggerModelResponse, err error) {
	resp = &modelpb.TriggerModelResponse{}

	namespaceID, modelID, err := parseModelName(req.GetName())
	if err != nil {
		return nil, err
	}

	params := triggerModelParams{
		namespaceID: namespaceID,
		modelID:     modelID,
		version:     "", // empty for latest
		taskInputs:  req.GetTaskInputs(),
	}

	resp.Task, resp.TaskOutputs, err = h.triggerModel(ctx, params)

	return resp, err
}

func (h *PublicHandler) triggerModel(ctx context.Context, params triggerModelParams) (commonpb.Task, []*structpb.Struct, error) {

	startTime := time.Now()

	logger, _ := logx.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, params.namespaceID)
	if err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}
	if err = authenticateUser(ctx, false); err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	pbModel, err := h.service.GetModelByID(ctx, ns, params.modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	modelUID, err := h.service.GetModelUIDByID(ctx, ns, params.modelID)
	if err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var version *datamodel.ModelVersion
	versionID := params.version

	if versionID == "" {
		version, err = h.service.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
		if err != nil {
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.NotFound, err.Error())
		}
	} else {
		version, err = h.service.GetModelVersionAdmin(ctx, modelUID, versionID)
		if err != nil {
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.NotFound, err.Error())
		}
	}

	logUUID, _ := uuid.NewV4()

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID:       requesterUID.String(),
		ModelID:            pbModel.Id,
		ModelUID:           modelUID.String(),
		Mode:               mgmtpb.Mode_MODE_SYNC,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	// Marshal task inputs for logging
	inputData := map[string]interface{}{
		"namespace_id": params.namespaceID,
		"model_id":     params.modelID,
		"version":      params.version,
		"task_inputs":  params.taskInputs,
	}
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	runLog, err := h.service.CreateModelRun(ctx, logUUID, modelUID, version.Version, inputJSON)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// write trigger/usage/metric datapoint
	var triggerErr error
	defer func(u *utils.UsageMetricData, startTime time.Time) {
		if err != nil && triggerErr == nil {
			_ = h.service.UpdateModelRunWithError(ctx, runLog, err)
		}
		u.ComputeTimeDuration = time.Since(startTime).Seconds()
		if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
			logger.Warn("usage/metric write failed")
		}
	}(usageData, startTime)

	// check whether model support batching or not. If not, raise an error
	if len(params.taskInputs) > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	for _, i := range params.taskInputs {
		i.Fields["data"].GetStructValue().Fields["model"] = structpb.NewStringValue(pbModel.Id)
		if err = datamodel.ValidateJSONSchema(datamodel.TasksJSONInputSchemaMap[pbModel.Task.String()], i, false); err != nil {
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	response, triggerErr := h.service.TriggerModelVersionByID(ctx, ns, params.modelID, version, inputJSON, pbModel.Task, runLog)
	if err != nil {
		var st *status.Status
		if strings.Contains(err.Error(), "failed to allocate memory") {
			st = status.New(codes.ResourceExhausted, "inference model error: Out of memory for running the model, maybe try with smaller batch size")
		} else {
			st = status.New(codes.FailedPrecondition, fmt.Sprintf("inference model error: %s", err.Error()))
		}
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return commonpb.Task_TASK_UNSPECIFIED, nil, st.Err()
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

	logger.Info("TriggerModelVersion",
		zap.Any("eventResource", fmt.Sprintf("userID: %s, modelID: %s, versionID: %s", ns.Name(), params.modelID, versionID)),
		zap.String("eventMessage", "TriggerModelVersion done"),
	)

	return pbModel.Task, response, nil
}

// TriggerAsyncModelVersion triggers a model for a given namespace.
func (h *PublicHandler) TriggerAsyncModelVersion(ctx context.Context, req *modelpb.TriggerAsyncModelVersionRequest) (resp *modelpb.TriggerAsyncModelVersionResponse, err error) {
	resp = &modelpb.TriggerAsyncModelVersionResponse{}

	namespaceID, modelID, version, err := parseModelVersionName(req.GetName())
	if err != nil {
		return nil, err
	}

	params := triggerModelParams{
		namespaceID: namespaceID,
		modelID:     modelID,
		version:     version,
		taskInputs:  req.GetTaskInputs(),
	}

	resp.Operation, err = h.triggerAsyncModel(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// TriggerAsyncModel triggers a model for a given namespace.
func (h *PublicHandler) TriggerAsyncModel(ctx context.Context, req *modelpb.TriggerAsyncModelRequest) (resp *modelpb.TriggerAsyncModelResponse, err error) {
	resp = &modelpb.TriggerAsyncModelResponse{}

	namespaceID, modelID, err := parseModelName(req.GetName())
	if err != nil {
		return nil, err
	}

	params := triggerModelParams{
		namespaceID: namespaceID,
		modelID:     modelID,
		version:     "", // empty for latest
		taskInputs:  req.GetTaskInputs(),
	}

	resp.Operation, err = h.triggerAsyncModel(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (h *PublicHandler) triggerAsyncModel(ctx context.Context, params triggerModelParams) (operation *longrunningpb.Operation, err error) {

	startTime := time.Now()

	logger, _ := logx.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, params.namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByID(ctx, ns, params.modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	modelUID, err := h.service.GetModelUIDByID(ctx, ns, params.modelID)
	if err != nil {
		return nil, err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		return nil, err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var version *datamodel.ModelVersion
	versionID := params.version

	if versionID == "" {
		version, err = h.service.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
	} else {
		version, err = h.service.GetModelVersionAdmin(ctx, modelUID, versionID)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
	}

	logUUID, _ := uuid.NewV4()

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID:       requesterUID.String(),
		ModelID:            pbModel.Id,
		ModelUID:           modelUID.String(),
		Mode:               mgmtpb.Mode_MODE_ASYNC,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	// Marshal task inputs for logging
	inputData := map[string]interface{}{
		"namespace_id": params.namespaceID,
		"model_id":     params.modelID,
		"version":      params.version,
		"task_inputs":  params.taskInputs,
	}
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	runLog, err := h.service.CreateModelRun(ctx, logUUID, modelUID, version.Version, inputJSON)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// write usage/metric datapoint
	defer func(u *utils.UsageMetricData, startTime time.Time) {
		if err != nil {
			_ = h.service.UpdateModelRunWithError(ctx, runLog, err)
		}
		if u.Status == mgmtpb.Status_STATUS_ERRORED {
			u.ComputeTimeDuration = time.Since(startTime).Seconds()
			if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
				logger.Warn("usage/metric write failed")
			}
		}
	}(usageData, startTime)

	// check whether model support batching or not. If not, raise an error
	if len(params.taskInputs) > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	for _, i := range params.taskInputs {
		i.Fields["data"].GetStructValue().Fields["model"] = structpb.NewStringValue(pbModel.Id)
		if err = datamodel.ValidateJSONSchema(datamodel.TasksJSONInputSchemaMap[pbModel.Task.String()], i, false); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	operation, err = h.service.TriggerAsyncModelVersionByID(ctx, ns, params.modelID, version, inputJSON, pbModel.Task, runLog)
	if err != nil {
		var st *status.Status
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st = status.New(codes.ResourceExhausted, "inference model error: Out of memory for running the model, maybe try with smaller batch size")
		} else {
			st = status.New(codes.FailedPrecondition, fmt.Sprintf("inference model error: %s", err.Error()))
		}
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return nil, st.Err()
	}

	// latest operation
	h.service.GetRedisClient().Set(
		ctx,
		fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", userUID, requesterUID, modelUID.String(), ""),
		operation.GetName(),
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)
	// latest version operation
	h.service.GetRedisClient().Set(
		ctx,
		fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", userUID, requesterUID, modelUID.String(), version.Version),
		operation.GetName(),
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	logger.Info("TriggerModelVersion",
		zap.Any("eventResource", fmt.Sprintf("userID: %s, modelID: %s, versionID: %s", ns.Name(), params.modelID, versionID)),
		zap.String("eventMessage", "TriggerModelVersion done"),
	)

	return operation, nil
}

// HandleTriggerMultipartForm handles the multipart form request.
func HandleTriggerMultipartForm(s service.Service, _ repository.Repository, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	startTime := time.Now()

	// inject header into ctx
	headers := map[string]string{}
	// inject header into ctx
	for key, value := range req.Header {
		if len(value) > 0 {
			headers[key] = value[0]
		}
	}
	md := metadata.New(headers)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	logger, _ := logx.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")

	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		return
	}

	if err := authenticateUser(ctx, false); err != nil {
		logger.Error(fmt.Sprintf("AuthenticatedUser Error: %s", err.Error()))
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			return
		}
	}

	namespaceID := strings.Split(pathParams["path"], "/")[1]
	modelID := strings.Split(pathParams["path"], "/")[3]

	ns, err := s.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		makeJSONResponse(w, 400, "Model path format error", "Model path format error")
		return
	}

	pbModel, err := s.GetModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		logger.Error(fmt.Sprintf("GetModelByID Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Model not found", "The model not found in server")
		return
	}

	modelUID, err := s.GetModelUIDByID(ctx, ns, modelID)
	if err != nil {
		logger.Error(fmt.Sprintf("GetModelUIDByID Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Model not found", "The model not found in server")
		return
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		return
	}

	modelDef, err := s.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		logger.Error(fmt.Sprintf("GetModelDefinition Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Model definition not found", "The model definition not found in server")
		return
	}

	var version *datamodel.ModelVersion
	if versionStr, ok := pathParams["version"]; !ok {
		version, err = s.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
		if err != nil {
			logger.Error(fmt.Sprintf("GetModelVersion Error: %s", err.Error()))
			makeJSONResponse(w, 404, "Version not found", "The model version not found in server")
			return
		}
	} else {
		version, err = s.GetModelVersionAdmin(ctx, modelUID, versionStr)
		if err != nil {
			logger.Error(fmt.Sprintf("GetModelVersion Error: %s", err.Error()))
			makeJSONResponse(w, 404, "Version not found", "The model version not found in server")
			return
		}
	}

	logUUID, _ := uuid.NewV4()

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID:       requesterUID.String(),
		ModelID:            pbModel.Id,
		ModelUID:           modelUID.String(),
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	varMap := map[string]any{}

	for k, v := range req.MultipartForm.Value {
		var b any
		unmarshalErr := json.Unmarshal([]byte(v[0]), &b)
		if unmarshalErr != nil {
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading variables from request %w", unmarshalErr))
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		varMap[k] = b
	}

	for k, v := range req.MultipartForm.File {
		file, err := v[0].Open()
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}

		byteContainer, err := io.ReadAll(file)
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		v := fmt.Sprintf("data:%s;base64,%s", v[0].Header.Get("Content-Type"), base64.StdEncoding.EncodeToString(byteContainer))
		varMap[k] = v
	}

	data := &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}
	for k, v := range varMap {
		structVal, err := structpb.NewValue(v)
		if err != nil {
			makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while parsing data from request %w", err))
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		data.Fields[k] = structVal
	}

	// Create a trigger request with the new name format
	inputReq := &modelpb.TriggerModelVersionRequest{
		Name:       fmt.Sprintf("namespaces/%s/models/%s/versions/%s", ns.NsID, modelID, version.Version),
		TaskInputs: []*structpb.Struct{data},
	}

	inputJSON, err := json.Marshal(inputReq)
	if err != nil {
		makeJSONResponse(w, 400, "Parser input error", err.Error())
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	runLog, err := s.CreateModelRun(ctx, logUUID, modelUID, version.Version, inputJSON)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		logger.Error("CreateModelRun in DB failed", zap.String("TriggerUID", logUUID.String()), zap.Error(err))
		makeJSONResponse(w, 500, "CreateModelRun in DB failed", "CreateModelRun in DB failed")
		return
	}

	// write usage/metric datapoint
	defer func(u *utils.UsageMetricData, startTime time.Time) {
		if err != nil {
			_ = s.UpdateModelRunWithError(ctx, runLog, err)
		}
		u.ComputeTimeDuration = time.Since(startTime).Seconds()
		if err := s.WriteNewDataPoint(ctx, usageData); err != nil {
			logger.Warn("usage/metric write failed")
		}
	}(usageData, startTime)

	data.Fields["data"].GetStructValue().Fields["model"] = structpb.NewStringValue(pbModel.Id)
	if err = datamodel.ValidateJSONSchema(datamodel.TasksJSONInputSchemaMap[pbModel.Task.String()], data, false); err != nil {
		makeJSONResponse(w, 400, "Invalid argument", fmt.Sprint("Error while parsing data from request %w", err))
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	var response []*structpb.Struct
	response, err = s.TriggerModelVersionByID(ctx, ns, modelID, version, inputJSON, pbModel.Task, runLog)
	if err != nil {
		var st *status.Status
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st = status.New(codes.ResourceExhausted, "inference model error: Out of memory for running the model, maybe try with smaller batch size")
		} else {
			st = status.New(codes.FailedPrecondition, fmt.Sprintf("inference model error: %s", err.Error()))
		}
		obj, _ := json.Marshal(st.Details())
		makeJSONResponse(w, 500, st.Message(), string(obj))
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(200)
	res, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseEnumNumbers:  false,
	}.Marshal(&modelpb.TriggerModelVersionBinaryFileUploadResponse{
		Task:        pbModel.Task,
		TaskOutputs: response,
	})
	if err != nil {
		makeJSONResponse(w, 500, "Error Predict Model", err.Error())
		return
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED
	if err = s.WriteNewDataPoint(ctx, usageData); err != nil {
		logger.Warn("usage and metric data write fail")
	}

	logger.Info("TriggerModelVersion",
		zap.Any("eventResource", pbModel.Id),
		zap.String("eventMessage", "TriggerModelVersion done"),
	)

	_, _ = w.Write(res)
}
