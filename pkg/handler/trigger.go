package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/instill-ai/x/sterr"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
)

func (h *PublicHandler) TriggerUserModel(ctx context.Context, req *modelpb.TriggerUserModelRequest) (*modelpb.TriggerUserModelResponse, error) {
	r, err := h.TriggerNamespaceModel(ctx, &modelpb.TriggerNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerUserModelResponse{
		Task:        r.Task,
		TaskOutputs: r.TaskOutputs,
	}, nil
}

func (h *PublicHandler) TriggerOrganizationModel(ctx context.Context, req *modelpb.TriggerOrganizationModelRequest) (*modelpb.TriggerOrganizationModelResponse, error) {
	r, err := h.TriggerNamespaceModel(ctx, &modelpb.TriggerNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerOrganizationModelResponse{
		Task:        r.Task,
		TaskOutputs: r.TaskOutputs,
	}, nil
}

func (h *PublicHandler) TriggerUserLatestModel(ctx context.Context, req *modelpb.TriggerUserLatestModelRequest) (*modelpb.TriggerUserLatestModelResponse, error) {
	r, err := h.TriggerNamespaceLatestModel(ctx, &modelpb.TriggerNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerUserLatestModelResponse{
		Task:        r.Task,
		TaskOutputs: r.TaskOutputs,
	}, nil
}

func (h *PublicHandler) TriggerOrganizationLatestModel(ctx context.Context, req *modelpb.TriggerOrganizationLatestModelRequest) (*modelpb.TriggerOrganizationLatestModelResponse, error) {
	r, err := h.TriggerNamespaceLatestModel(ctx, &modelpb.TriggerNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerOrganizationLatestModelResponse{
		Task:        r.Task,
		TaskOutputs: r.TaskOutputs,
	}, nil
}

type TriggerNamespaceModelRequestInterface interface {
	GetNamespaceId() string
	GetModelId() string
	GetVersion() string
	GetTaskInputs() []*modelpb.TaskInput
}

func (h *PublicHandler) TriggerNamespaceModel(ctx context.Context, req *modelpb.TriggerNamespaceModelRequest) (resp *modelpb.TriggerNamespaceModelResponse, err error) {
	resp = &modelpb.TriggerNamespaceModelResponse{}

	r := &modelpb.TriggerNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
		Version:     req.GetVersion(),
		TaskInputs:  req.GetTaskInputs(),
	}

	resp.Task, resp.TaskOutputs, err = h.triggerNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) TriggerNamespaceLatestModel(ctx context.Context, req *modelpb.TriggerNamespaceLatestModelRequest) (resp *modelpb.TriggerNamespaceLatestModelResponse, err error) {
	resp = &modelpb.TriggerNamespaceLatestModelResponse{}

	r := &modelpb.TriggerNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
		TaskInputs:  req.GetTaskInputs(),
	}

	resp.Task, resp.TaskOutputs, err = h.triggerNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) triggerNamespaceModel(ctx context.Context, req TriggerNamespaceModelRequestInterface) (commonpb.Task, []*modelpb.TaskOutput, error) {

	startTime := time.Now()
	eventName := "TriggerNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}
	if err = authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, req.GetModelId(), modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonpb.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var version *datamodel.ModelVersion
	versionID := req.GetVersion()
	modelUID := uuid.FromStringOrNil(pbModel.Uid)

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

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		Mode:               mgmtpb.Mode_MODE_SYNC,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	// write usage/metric datapoint
	defer func(u *utils.UsageMetricData, startTime time.Time) {
		u.ComputeTimeDuration = time.Since(startTime).Seconds()
		if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
			logger.Warn("usage/metric write failed")
		}
	}(usageData, startTime)

	var parsedInput any
	var lenInputs = 1
	switch pbModel.Task {
	case commonpb.Task_TASK_CLASSIFICATION,
		commonpb.Task_TASK_DETECTION,
		commonpb.Task_TASK_INSTANCE_SEGMENTATION,
		commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
		commonpb.Task_TASK_OCR,
		commonpb.Task_TASK_KEYPOINT,
		commonpb.Task_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		parsedInput = imageInput
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textToImage
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		imageToImage, err := parseImageToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = imageToImage
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnswering, err := parseVisualQuestionAnsweringRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		parsedInput = visualQuestionAnswering
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChat, err := parseTexGenerationChatRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textGenerationChat
	case commonpb.Task_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	parsedInputJSON, err := json.Marshal(parsedInput)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return commonpb.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	response, err := h.service.TriggerNamespaceModelByID(ctx, ns, req.GetModelId(), version, parsedInputJSON, pbModel.Task, logUUID.String())
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
			"ray server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"ray server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return commonpb.Task_TASK_UNSPECIFIED, nil, st.Err()
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel.Name),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel.Task, response, nil
}

func (h *PublicHandler) TriggerAsyncUserModel(ctx context.Context, req *modelpb.TriggerAsyncUserModelRequest) (*modelpb.TriggerAsyncUserModelResponse, error) {
	r, err := h.TriggerAsyncNamespaceModel(ctx, &modelpb.TriggerAsyncNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerAsyncUserModelResponse{
		Operation: r.Operation,
	}, nil
}

func (h *PublicHandler) TriggerAsyncOrganizationModel(ctx context.Context, req *modelpb.TriggerAsyncOrganizationModelRequest) (*modelpb.TriggerAsyncOrganizationModelResponse, error) {
	r, err := h.TriggerAsyncNamespaceModel(ctx, &modelpb.TriggerAsyncNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerAsyncOrganizationModelResponse{
		Operation: r.Operation,
	}, nil
}

func (h *PublicHandler) TriggerAsyncUserLatestModel(ctx context.Context, req *modelpb.TriggerAsyncUserLatestModelRequest) (*modelpb.TriggerAsyncUserLatestModelResponse, error) {
	r, err := h.TriggerAsyncNamespaceLatestModel(ctx, &modelpb.TriggerAsyncNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerAsyncUserLatestModelResponse{
		Operation: r.Operation,
	}, nil
}

func (h *PublicHandler) TriggerAsyncOrganizationLatestModel(ctx context.Context, req *modelpb.TriggerAsyncOrganizationLatestModelRequest) (*modelpb.TriggerAsyncOrganizationLatestModelResponse, error) {
	r, err := h.TriggerAsyncNamespaceLatestModel(ctx, &modelpb.TriggerAsyncNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		TaskInputs:  req.TaskInputs,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.TriggerAsyncOrganizationLatestModelResponse{
		Operation: r.Operation,
	}, nil
}

type TriggerAsyncNamespaceModelRequestInterface interface {
	protoreflect.ProtoMessage
	GetNamespaceId() string
	GetModelId() string
	GetVersion() string
	GetTaskInputs() []*modelpb.TaskInput
}

func (h *PublicHandler) TriggerAsyncNamespaceModel(ctx context.Context, req *modelpb.TriggerAsyncNamespaceModelRequest) (resp *modelpb.TriggerAsyncNamespaceModelResponse, err error) {
	resp = &modelpb.TriggerAsyncNamespaceModelResponse{}

	r := &modelpb.TriggerAsyncNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
		Version:     req.GetVersion(),
		TaskInputs:  req.GetTaskInputs(),
	}

	resp.Operation, err = h.triggerAsyncNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) TriggerAsyncNamespaceLatestModel(ctx context.Context, req *modelpb.TriggerAsyncNamespaceLatestModelRequest) (resp *modelpb.TriggerAsyncNamespaceLatestModelResponse, err error) {
	resp = &modelpb.TriggerAsyncNamespaceLatestModelResponse{}

	r := &modelpb.TriggerAsyncNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
		TaskInputs:  req.GetTaskInputs(),
	}

	resp.Operation, err = h.triggerAsyncNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) triggerAsyncNamespaceModel(ctx context.Context, req TriggerAsyncNamespaceModelRequestInterface) (operation *longrunningpb.Operation, err error) {

	eventName := "TriggerAsyncNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	startTime := time.Now()
	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, req.GetModelId(), modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var version *datamodel.ModelVersion
	versionID := req.GetVersion()
	if versionID == "" {
		version, err = h.service.GetRepository().GetLatestModelVersionByModelUID(ctx, uuid.FromStringOrNil(pbModel.Uid))
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
	} else {
		version, err = h.service.GetModelVersionAdmin(ctx, uuid.FromStringOrNil(pbModel.Uid), versionID)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
	}

	inputJSON, err := json.Marshal(req.GetTaskInputs())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))

	// TODO: temporary solution to store input json for latest operation
	inputRequestJSON, err := protojson.Marshal(req)
	if err != nil {
		return nil, err
	}
	h.service.GetRedisClient().Set(
		ctx,
		fmt.Sprintf("model_trigger_input:%s:%s", userUID.String(), pbModel.Uid),
		inputRequestJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		Mode:               mgmtpb.Mode_MODE_ASYNC,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	// write usage/metric datapoint
	defer func(u *utils.UsageMetricData, startTime time.Time) {
		if u.Status == mgmtpb.Status_STATUS_ERRORED {
			u.ComputeTimeDuration = time.Since(startTime).Seconds()
			if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
				logger.Warn("usage/metric write failed")
			}
		}
	}(usageData, startTime)

	var parsedInput any
	var lenInputs = 1
	switch pbModel.Task {
	case commonpb.Task_TASK_CLASSIFICATION,
		commonpb.Task_TASK_DETECTION,
		commonpb.Task_TASK_INSTANCE_SEGMENTATION,
		commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
		commonpb.Task_TASK_OCR,
		commonpb.Task_TASK_KEYPOINT,
		commonpb.Task_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		parsedInput = imageInput
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textToImage
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		imageToImage, err := parseImageToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = imageToImage
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnswering, err := parseVisualQuestionAnsweringRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		parsedInput = visualQuestionAnswering
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChat, err := parseTexGenerationChatRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textGenerationChat
	case commonpb.Task_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		parsedInput = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if !doSupportBatch {
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			return nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
		}
	}

	parsedInputJSON, err := json.Marshal(parsedInput)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	operation, err = h.service.TriggerAsyncNamespaceModelByID(ctx, ns, req.GetModelId(), version, inputJSON, parsedInputJSON, pbModel.Task, logUUID.String())
	if err != nil {
		span.SetStatus(1, err.Error())
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		return nil, err
	}

	// TODO: temporary solution to store output json
	h.service.GetRedisClient().Set(
		ctx,
		fmt.Sprintf("model_trigger_output_key:%s:%s", userUID, pbModel.Uid),
		operation.GetName(),
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return operation, nil
}

func inferModelByUpload(s service.Service, _ repository.Repository, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	startTime := time.Now()
	eventName := "InferModelByUpload"

	ctx, span := tracer.Start(req.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// inject header into ctx
	headers := map[string]string{}
	// inject header into ctx
	for key, value := range req.Header {
		if len(value) > 0 {
			headers[key] = value[0]
		}
	}
	md := metadata.New(headers)
	ctx = metadata.NewIncomingContext(ctx, md)

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")

	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		span.SetStatus(1, "")
		return
	}

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	if err := authenticateUser(ctx, false); err != nil {
		logger.Error(fmt.Sprintf("AuthenticatedUser Error: %s", err.Error()))
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			span.SetStatus(1, "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			span.SetStatus(1, "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			return
		}
	}

	namespaceID := strings.Split(pathParams["path"], "/")[1]
	modelID := strings.Split(pathParams["path"], "/")[3]

	ns, err := s.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		makeJSONResponse(w, 400, "Model path format error", "Model path format error")
		span.SetStatus(1, "Model path format error")
		return
	}

	pbModel, err := s.GetNamespaceModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		logger.Error(fmt.Sprintf("GetNamespaceModelByID Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Model not found", "The model not found in server")
		span.SetStatus(1, "The model not found in server")
		return
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return
	}

	modelDef, err := s.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		logger.Error(fmt.Sprintf("GetModelDefinition Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Model definition not found", "The model definition not found in server")
		span.SetStatus(1, "The model definition not found in server")
		return
	}

	version, err := s.GetModelVersionAdmin(ctx, uuid.FromStringOrNil(pbModel.Uid), pathParams["version"])
	if err != nil {
		logger.Error(fmt.Sprintf("GetModelVersion Error: %s", err.Error()))
		makeJSONResponse(w, 404, "Version not found", "The model version not found in server")
		span.SetStatus(1, "The model version not found in server")
		return
	}

	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	usageData := &utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:            userUID,
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          pbModel.Task,
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 400, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	var parsedInput any
	var lenInputs = 1
	switch pbModel.Task {
	case commonpb.Task_TASK_CLASSIFICATION,
		commonpb.Task_TASK_DETECTION,
		commonpb.Task_TASK_INSTANCE_SEGMENTATION,
		commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
		commonpb.Task_TASK_OCR,
		commonpb.Task_TASK_KEYPOINT,
		commonpb.Task_TASK_UNSPECIFIED:
		imageInput, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			makeJSONResponse(w, 400, "File Input Error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		lenInputs = len(imageInput)
		parsedInput = imageInput
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseImageFormDataTextToImageInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		parsedInput = textToImage
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		imageToImage, err := parseImageFormDataImageToImageInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		parsedInput = imageToImage
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnswering, err := parseTextFormDataVisualQuestionAnsweringInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		parsedInput = visualQuestionAnswering
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChat, err := parseTextFormDataTextGenerationChatInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		parsedInput = textGenerationChat
	case commonpb.Task_TASK_TEXT_GENERATION:
		textGeneration, err := parseTextFormDataTextGenerationInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		parsedInput = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		doSupportBatch, err := utils.DoSupportBatch()
		if err != nil {
			makeJSONResponse(w, 400, "Batching Support Error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		if !doSupportBatch {
			makeJSONResponse(w, 400, "Batching Support Error", "The model do not support batching, so could not make inference with multiple images")
			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
			usageData.Status = mgmtpb.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
	}

	parsedInputJSON, err := json.Marshal(parsedInput)
	if err != nil {
		makeJSONResponse(w, 400, "Parser input error", err.Error())
		span.SetStatus(1, err.Error())
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	var response []*modelpb.TaskOutput
	response, err = s.TriggerNamespaceModelByID(ctx, ns, modelID, version, parsedInputJSON, pbModel.Task, logUUID.String())
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
		obj, _ := json.Marshal(st.Details())
		makeJSONResponse(w, 500, st.Message(), string(obj))
		span.SetStatus(1, st.Message())
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(200)
	res, err := utils.MarshalOptions.Marshal(&modelpb.TriggerUserModelBinaryFileUploadResponse{
		Task:        pbModel.Task,
		TaskOutputs: response,
	})
	if err != nil {
		makeJSONResponse(w, 500, "Error Predict Model", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED
	if err := s.WriteNewDataPoint(ctx, usageData); err != nil {
		logger.Warn("usage and metric data write fail")
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	_, _ = w.Write(res)

}

func HandleTriggerModelByUpload(s service.Service, repo repository.Repository, w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelByUpload(s, repo, w, r, pathParams)
}
