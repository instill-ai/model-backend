package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func createContainerizedModel(s service.Service, ctx context.Context, model *modelPB.Model, ns resource.Namespace, modelDefinition *datamodel.ModelDefinition) (*modelPB.Model, error) {

	eventName := "CreateContainerizedModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	var modelConfig datamodel.ContainerizedModelConfiguration
	b, err := model.GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.Model{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.Model{}, status.Errorf(codes.InvalidArgument, err.Error())
	}

	bModelConfig, _ := json.Marshal(modelConfig)

	containerizedModel, err := s.PBToDBModel(ctx, ns, model)
	if err != nil {
		return nil, err
	}
	containerizedModel.Configuration = bModelConfig
	containerizedModel.ModelDefinitionUID = modelDefinition.UID

	// modelMeta := utils.ModelMeta{
	// 	Tags: []string{"Containerized", "Experimental"},
	// 	Task: model.Task.String(),
	// }

	containerizedModel.Task = datamodel.ModelTask(utils.Tasks[model.Task.String()])
	// if val, ok := utils.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
	// 	containerizedModel.Task = datamodel.ModelTask(val)
	// } else {
	// 	if modelMeta.Task != "" {
	// 		st, err := sterr.CreateErrorResourceInfo(
	// 			codes.FailedPrecondition,
	// 			"[handler] create a model error: unsupported task",
	// 			"request body",
	// 			"request body contains unsupported task",
	// 			"",
	// 			"",
	// 		)
	// 		if err != nil {
	// 			logger.Error(err.Error())
	// 		}
	// 		span.SetStatus(1, st.Err().Error())
	// 		return &modelPB.Model{}, st.Err()
	// 	} else {
	// 		containerizedModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
	// 	}
	// }

	// TODO: properly support batch inference
	maxBatchSize := 0
	allowedMaxBatchSize := utils.GetSupportedBatchSize(containerizedModel.Task)

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
		return &modelPB.Model{}, st.Err()
	}

	if err := s.CreateNamespaceModel(ctx, ns, containerizedModel); err != nil {
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
		span.SetStatus(1, st.Err().Error())
		return &modelPB.Model{}, st.Err()
	}

	model, _ = s.GetNamespaceModelByID(ctx, ns, model.Id, modelPB.View_VIEW_FULL)

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(containerizedModel),
	)))

	return model, nil
}
