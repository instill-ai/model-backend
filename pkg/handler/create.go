package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/x/sterr"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func createContainerizedModel(s service.Service, ctx context.Context, ns resource.Namespace, modelDefinition *datamodel.ModelDefinition, model *modelpb.Model) (*modelpb.CreateNamespaceModelResponse, error) {

	eventName := "CreateContainerizedModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	if err := s.CreateNamespaceModel(ctx, ns, modelDefinition, model); err != nil {
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
		return &modelpb.CreateNamespaceModelResponse{}, st.Err()
	}

	model, err := s.GetNamespaceModelByID(ctx, ns, model.Id, modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

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
	)))

	return &modelpb.CreateNamespaceModelResponse{
		Model: model,
	}, nil
}
