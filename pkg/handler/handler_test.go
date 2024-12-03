package handler_test

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/model-backend/pkg/handler"
	"github.com/instill-ai/model-backend/pkg/mock"
	"github.com/instill-ai/model-backend/pkg/utils"

	healthcheckpb "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

const NAMESPACE = "instill-ai"

func TestReadiness(t *testing.T) {
	t.Run("Readiness", func(t *testing.T) {
		mc := minimock.NewController(t)

		mockRay := mock.NewRayMock(mc)
		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, utils.Testing, true)
		mockRay.IsRayServerReadyMock.Times(1).Expect(ctxWithValue).Return(true)

		h := handler.NewPublicHandler(ctxWithValue, nil, mockRay, nil)
		readyRes, err := h.Readiness(ctxWithValue, &modelpb.ReadinessRequest{})

		assert.NoError(t, err)
		assert.Equal(t, readyRes.HealthCheckResponse.Status, healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING)
	})
}

func TestLiveness(t *testing.T) {
	t.Run("Liveness", func(t *testing.T) {
		mc := minimock.NewController(t)

		mockRay := mock.NewRayMock(mc)
		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, utils.Testing, true)
		mockRay.IsRayServerReadyMock.Times(1).Expect(ctxWithValue).Return(true)

		h := handler.NewPublicHandler(ctxWithValue, nil, mockRay, nil)

		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		// defer cancel()
		liveRes, err := h.Liveness(ctxWithValue, &modelpb.LivenessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, liveRes.HealthCheckResponse.Status, healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING)
	})
}
