package handler

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/resource"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (h *PublicHandler) GetModelOperation(ctx context.Context, req *modelpb.GetModelOperationRequest) (*modelpb.GetModelOperationResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelOperation",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	operationID, err := resource.GetOperationID(req.OperationId)
	if err != nil {
		return &modelpb.GetModelOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationID)
	if err != nil {
		return &modelpb.GetModelOperationResponse{}, err
	}

	return &modelpb.GetModelOperationResponse{
		Operation: operation,
	}, nil
}

func (h *PublicHandler) GetUserLatestModelOperation(ctx context.Context, req *modelpb.GetUserLatestModelOperationRequest) (*modelpb.GetUserLatestModelOperationResponse, error) {
	r, err := h.GetNamespaceLatestModelOperation(ctx, &modelpb.GetNamespaceLatestModelOperationRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.GetUserLatestModelOperationResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) GetOrganizationLatestModelOperation(ctx context.Context, req *modelpb.GetOrganizationLatestModelOperationRequest) (*modelpb.GetOrganizationLatestModelOperationResponse, error) {
	r, err := h.GetNamespaceLatestModelOperation(ctx, &modelpb.GetNamespaceLatestModelOperationRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.GetOrganizationLatestModelOperationResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) GetNamespaceLatestModelOperation(ctx context.Context, req *modelpb.GetNamespaceLatestModelOperationRequest) (*modelpb.GetNamespaceLatestModelOperationResponse, error) {
	eventName := "GetNamespaceLatestModelOperation"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			ctx,
			span,
			logUUID.String(),
			eventName,
			custom_otel.SetEventResource(req.GetModelId()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return nil, err
	}

	operation, err := h.service.GetNamespaceLatestModelOperation(ctx, ns, req.GetModelId(), req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &modelpb.GetNamespaceLatestModelOperationResponse{Operation: operation}, nil
}
