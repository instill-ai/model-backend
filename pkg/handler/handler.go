package handler

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/usage"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var tracer = otel.Tracer("model-backend.public-handler.tracer")

type PublicHandler struct {
	modelpb.UnimplementedModelPublicServiceServer
	service           service.Service
	ray               ray.Ray
	modelUsageHandler usage.ModelUsageHandler
}

func NewPublicHandler(ctx context.Context, s service.Service, r ray.Ray, h usage.ModelUsageHandler) modelpb.ModelPublicServiceServer {
	datamodel.InitJSONSchema(ctx)
	if h == nil {
		h = usage.NewNoopModelUsageHandler()
	}
	return &PublicHandler{
		service:           s,
		ray:               r,
		modelUsageHandler: h,
	}
}

// GetService returns the service
func (h *PublicHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PublicHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PublicHandler) Liveness(ctx context.Context, pb *modelpb.LivenessRequest) (*modelpb.LivenessResponse, error) {
	if !h.ray.IsRayServerReady(ctx) {
		return &modelpb.LivenessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelpb.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, pb *modelpb.ReadinessRequest) (*modelpb.ReadinessResponse, error) {
	if !h.ray.IsRayServerReady(ctx) {
		return &modelpb.ReadinessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelpb.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

type PrivateHandler struct {
	modelpb.UnimplementedModelPrivateServiceServer
	service service.Service
}

func NewPrivateHandler(ctx context.Context, s service.Service) modelpb.ModelPrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PrivateHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PrivateHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PrivateHandler) SetService(s service.Service) {
	h.service = s
}
