package rpc

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	model "github.com/instill-ai/protogen-go/model/v1alpha"
	"github.com/stretchr/testify/assert"
)

const NAMESPACE = "local-user"

func TestModelService_Readiness(t *testing.T) {
	t.Run("Readiness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockModelService := NewMockModelService(ctrl)
		mockTritonService := NewMockTritonService(ctrl)
		rpcService := serviceHandlers{
			modelService:  mockModelService,
			tritonService: mockTritonService,
		}
		mockTritonService.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		readyRes, err := rpcService.Readiness(ctx, &model.ReadinessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, readyRes.Status, model.ReadinessResponse_SERVING_STATUS_SERVING)
	})
}

func TestModelService_Liveness(t *testing.T) {
	t.Run("Liveness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockModelService := NewMockModelService(ctrl)
		mockTritonService := NewMockTritonService(ctrl)

		rpcService := serviceHandlers{
			modelService:  mockModelService,
			tritonService: mockTritonService,
		}
		mockTritonService.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		liveRes, err := rpcService.Liveness(ctx, &model.LivenessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, liveRes.Status, model.LivenessResponse_SERVING_STATUS_SERVING)
	})
}
