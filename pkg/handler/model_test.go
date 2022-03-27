package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gomock "github.com/golang/mock/gomock"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

const NAMESPACE = "local-user"

func TestModelService_Readiness(t *testing.T) {
	t.Run("Readiness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockModelService := NewMockModelService(ctrl)
		mockTritonService := NewMockTritonService(ctrl)
		rpcService := modelServiceHandler{
			modelService:  mockModelService,
			tritonService: mockTritonService,
		}
		mockTritonService.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		readyRes, err := rpcService.Readiness(ctx, &modelPB.ReadinessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, readyRes.Status, modelPB.ReadinessResponse_SERVING_STATUS_SERVING)
	})
}

func TestModelService_Liveness(t *testing.T) {
	t.Run("Liveness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockModelService := NewMockModelService(ctrl)
		mockTritonService := NewMockTritonService(ctrl)

		rpcService := modelServiceHandler{
			modelService:  mockModelService,
			tritonService: mockTritonService,
		}
		mockTritonService.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		liveRes, err := rpcService.Liveness(ctx, &modelPB.LivenessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, liveRes.Status, modelPB.LivenessResponse_SERVING_STATUS_SERVING)
	})
}
