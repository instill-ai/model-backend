package handler

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/instill-ai/model-backend/internal/resource"

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
