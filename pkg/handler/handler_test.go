package handler_test

//go:generate mockgen -destination mock_triton_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/internal/triton Triton
//go:generate mockgen -destination mock_service_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/service Service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gomock "github.com/golang/mock/gomock"

	"github.com/instill-ai/model-backend/pkg/handler"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

const NAMESPACE = "local-user"

func TestReadiness(t *testing.T) {
	t.Run("Readiness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockService := NewMockService(ctrl)
		mockTriton := NewMockTriton(ctrl)
		h := handler.NewHandler(mockService, mockTriton)

		mockTriton.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		readyRes, err := h.Readiness(ctx, &modelPB.ReadinessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, readyRes.Status, modelPB.ReadinessResponse_SERVING_STATUS_SERVING)
	})
}

func TestLiveness(t *testing.T) {
	t.Run("Liveness", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockService := NewMockService(ctrl)
		mockTriton := NewMockTriton(ctrl)
		h := handler.NewHandler(mockService, mockTriton)

		mockTriton.
			EXPECT().
			IsTritonServerReady().
			Return(true)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
		defer cancel()
		liveRes, err := h.Liveness(ctx, &modelPB.LivenessRequest{})
		assert.NoError(t, err)
		assert.Equal(t, liveRes.Status, modelPB.LivenessResponse_SERVING_STATUS_SERVING)
	})
}
