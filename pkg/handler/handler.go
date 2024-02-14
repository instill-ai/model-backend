package handler

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/service"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var tracer = otel.Tracer("model-backend.public-handler.tracer")

type PublicHandler struct {
	modelPB.UnimplementedModelPublicServiceServer
	service service.Service
	ray     ray.Ray
}

func NewPublicHandler(ctx context.Context, s service.Service, r ray.Ray) modelPB.ModelPublicServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PublicHandler{
		service: s,
		ray:     r,
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

func (h *PublicHandler) Liveness(ctx context.Context, pb *modelPB.LivenessRequest) (*modelPB.LivenessResponse, error) {
	if !h.ray.IsRayServerReady(ctx) {
		return &modelPB.LivenessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelPB.LivenessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, pb *modelPB.ReadinessRequest) (*modelPB.ReadinessResponse, error) {
	if !h.ray.IsRayServerReady(ctx) {
		return &modelPB.ReadinessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}

	return &modelPB.ReadinessResponse{HealthCheckResponse: &healthcheckPB.HealthCheckResponse{Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING}}, nil
}

type PrivateHandler struct {
	modelPB.UnimplementedModelPrivateServiceServer
	service service.Service
}

func NewPrivateHandler(ctx context.Context, s service.Service) modelPB.ModelPrivateServiceServer {
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
