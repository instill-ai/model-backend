package handler

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/pkg/resource"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	logx "github.com/instill-ai/x/log"
)

// parseOperationModelName parses a model name for operation requests.
// Format: namespaces/{namespace}/models/{model}
func parseOperationModelName(name string) (namespaceID, modelID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "namespaces" || parts[2] != "models" {
		return "", "", status.Errorf(codes.InvalidArgument, "invalid model name format: %s", name)
	}
	return parts[1], parts[3], nil
}

// parseOperationModelVersionName parses a model version name for operation requests.
// Format: namespaces/{namespace}/models/{model}/versions/{version}
func parseOperationModelVersionName(name string) (namespaceID, modelID, version string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 6 || parts[0] != "namespaces" || parts[2] != "models" || parts[4] != "versions" {
		return "", "", "", status.Errorf(codes.InvalidArgument, "invalid model version name format: %s", name)
	}
	return parts[1], parts[3], parts[5], nil
}

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

// GetNamespaceLatestModelOperation returns the latest model operation for a given namespace.
func (h *PublicHandler) GetNamespaceLatestModelOperation(ctx context.Context, req *modelpb.GetNamespaceLatestModelOperationRequest) (*modelpb.GetNamespaceLatestModelOperationResponse, error) {

	logger, _ := logx.GetZapLogger(ctx)

	namespaceID, modelID, err := parseOperationModelName(req.GetName())
	if err != nil {
		return nil, err
	}

	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		logger.Info("GetNamespaceLatestModelOperation",
			zap.Any("eventResource", modelID),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	operation, err := h.service.GetNamespaceLatestModelOperation(ctx, ns, modelID, req.GetView())
	if err != nil {
		logger.Info("GetNamespaceLatestModelOperation",
			zap.Any("eventResource", modelID),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	return &modelpb.GetNamespaceLatestModelOperationResponse{Operation: operation}, nil
}

// GetNamespaceModelOperation returns the model operation for a given namespace and model ID.
func (h *PublicHandler) GetNamespaceModelOperation(ctx context.Context, req *modelpb.GetNamespaceModelOperationRequest) (*modelpb.GetNamespaceModelOperationResponse, error) {

	logger, _ := logx.GetZapLogger(ctx)

	namespaceID, modelID, version, err := parseOperationModelVersionName(req.GetName())
	if err != nil {
		return nil, err
	}

	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		logger.Info("GetNamespaceModelOperation",
			zap.Any("eventResource", modelID),
			zap.String("errorMessage", err.Error()),
		)
		return nil, err
	}

	operation, err := h.service.GetNamespaceModelOperation(ctx, ns, modelID, version, req.GetView())
	if err != nil {
		return nil, err
	}

	return &modelpb.GetNamespaceModelOperationResponse{Operation: operation}, nil
}
