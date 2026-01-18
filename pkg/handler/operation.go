package handler

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/instill-ai/model-backend/pkg/resource"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	logx "github.com/instill-ai/x/log"
)

// GetModelOperation returns the operation details for a given model operation ID.
func (h *PublicHandler) GetModelOperation(ctx context.Context, req *modelpb.GetModelOperationRequest) (*modelpb.GetModelOperationResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelOperation",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	workflowID, err := resource.GetWorkflowID(req.OperationId)
	if err != nil {
		return &modelpb.GetModelOperationResponse{}, err
	}

	operation, err := h.service.GetOperation(ctx, workflowID)
	if err != nil {
		return &modelpb.GetModelOperationResponse{}, err
	}

	return &modelpb.GetModelOperationResponse{
		Operation: operation,
	}, nil
}

// GetUserLatestModelOperation returns the latest model operation for a given user.
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

// GetOrganizationLatestModelOperation returns the latest model operation for a given organization.
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

// GetNamespaceLatestModelOperation returns the latest model operation for a given namespace.
func (h *PublicHandler) GetNamespaceLatestModelOperation(ctx context.Context, req *modelpb.GetNamespaceLatestModelOperationRequest) (*modelpb.GetNamespaceLatestModelOperationResponse, error) {

	logger, _ := logx.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		logger.Info("GetNamespaceLatestModelOperation",
			zap.Any("eventResource", req.GetModelId()),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	operation, err := h.service.GetNamespaceLatestModelOperation(ctx, ns, req.GetModelId(), req.GetView())
	if err != nil {
		logger.Info("GetNamespaceLatestModelOperation",
			zap.Any("eventResource", req.GetModelId()),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	return &modelpb.GetNamespaceLatestModelOperationResponse{Operation: operation}, nil
}

// GetNamespaceModelOperation returns the model operation for a given namespace and model ID.
func (h *PublicHandler) GetNamespaceModelOperation(ctx context.Context, req *modelpb.GetNamespaceModelOperationRequest) (*modelpb.GetNamespaceModelOperationResponse, error) {

	logger, _ := logx.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		logger.Info("GetNamespaceModelOperation",
			zap.Any("eventResource", req.GetModelId()),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	operation, err := h.service.GetNamespaceModelOperation(ctx, ns, req.GetModelId(), req.GetVersion(), req.GetView())
	if err != nil {
		return nil, err
	}

	return &modelpb.GetNamespaceModelOperationResponse{Operation: operation}, nil
}
