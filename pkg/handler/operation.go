package handler

import (
	"context"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"go.opentelemetry.io/otel/trace"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/resource"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (h *PublicHandler) GetModelOperation(ctx context.Context, req *modelPB.GetModelOperationRequest) (*modelPB.GetModelOperationResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelOperation",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	operationID, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationID)
	if err != nil {
		return &modelPB.GetModelOperationResponse{}, err
	}

	return &modelPB.GetModelOperationResponse{
		Operation: operation,
	}, nil
}

type GetNamespaceLatestModelOperationRequestInterface interface {
	GetName() string
	GetView() modelPB.View
}

func (h *PublicHandler) GetUserLatestModelOperation(ctx context.Context, req *modelPB.GetUserLatestModelOperationRequest) (resp *modelPB.GetUserLatestModelOperationResponse, err error) {

	resp = &modelPB.GetUserLatestModelOperationResponse{}

	resp.Operation, err = h.getNamespaceLatestModelOperation(ctx, req)

	return resp, err
}

func (h *PublicHandler) GetOrganizationLatestModelOperation(ctx context.Context, req *modelPB.GetOrganizationLatestModelOperationRequest) (resp *modelPB.GetOrganizationLatestModelOperationResponse, err error) {

	resp = &modelPB.GetOrganizationLatestModelOperationResponse{}

	resp.Operation, err = h.getNamespaceLatestModelOperation(ctx, req)

	return resp, err
}

func (h *PublicHandler) getNamespaceLatestModelOperation(ctx context.Context, req GetNamespaceLatestModelOperationRequestInterface) (*longrunningpb.Operation, error) {
	eventName := "GetNamespaceLatestModelOperation"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
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
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return nil, err
	}

	operation, err := h.service.GetNamespaceLatestModelOperation(ctx, ns, modelID, req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return operation, nil
}
