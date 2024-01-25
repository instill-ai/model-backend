package handler

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var tracer = otel.Tracer("model-backend.public-handler.tracer")

type PublicHandler struct {
	modelPB.UnimplementedModelPublicServiceServer
	service service.Service
	triton  triton.Triton
	ray     ray.Ray
}

func NewPublicHandler(ctx context.Context, s service.Service, t triton.Triton, r ray.Ray) modelPB.ModelPublicServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PublicHandler{
		service: s,
		triton:  t,
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
	if !h.triton.IsTritonServerReady(ctx) {
		return &modelPB.LivenessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}
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
	if !h.triton.IsTritonServerReady(ctx) {
		return &modelPB.ReadinessResponse{
			HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
				Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_NOT_SERVING,
			},
		}, nil
	}
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
	triton  triton.Triton
}

func NewPrivateHandler(ctx context.Context, s service.Service, t triton.Triton) modelPB.ModelPrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PrivateHandler{
		service: s,
		triton:  t,
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
