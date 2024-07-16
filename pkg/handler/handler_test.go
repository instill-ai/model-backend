package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/handler"
	"github.com/instill-ai/model-backend/pkg/mock"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func TestReadiness(t *testing.T) {
	t.Run("Readiness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), constant.RunLevelFlag, constant.LevelUnitTest), time.Second*1000)
		defer cancel()

		mockService := mock.NewMockService(ctrl)
		mockRay := mock.NewMockRay(ctrl)
		mockRay.EXPECT().IsRayServerReady(ctx).Return(true).Times(1)
		h := handler.NewPublicHandler(ctx, mockService, mockRay, nil)

		readyRes, err := h.Readiness(ctx, &modelPB.ReadinessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, readyRes.HealthCheckResponse.Status, healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING)
	})
}

func TestLiveness(t *testing.T) {
	t.Run("Liveness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), constant.RunLevelFlag, constant.LevelUnitTest), time.Second*1000)
		defer cancel()

		mockService := mock.NewMockService(ctrl)
		mockRay := mock.NewMockRay(ctrl)
		mockRay.EXPECT().IsRayServerReady(ctx).Return(true).Times(1)
		h := handler.NewPublicHandler(ctx, mockService, mockRay, nil)

		liveRes, err := h.Liveness(ctx, &modelPB.LivenessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, liveRes.HealthCheckResponse.Status, healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING)
	})
}
