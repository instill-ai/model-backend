package handler_test

//go:generate mockgen -destination mock_service_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/service Service

const NAMESPACE = "instill-ai"

// func TestReadiness(t *testing.T) {
// 	t.Run("Readiness", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		mockService := NewMockService(ctrl)
// 		mockTriton := NewMockTriton(ctrl)
// 		h := handler.NewHandler(mockService, mockTriton)

// 		mockTriton.
// 			EXPECT().
// 			IsTritonServerReady().
// 			Return(true)

// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
// 		defer cancel()
// 		readyRes, err := h.Readiness(ctx, &modelPB.ReadinessRequest{})
// 		assert.NoError(t, err)
// 		assert.Equal(t, readyRes.HealthCheckResponse.Status, healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING)
// 	})
// }

// func TestLiveness(t *testing.T) {
// 	t.Run("Liveness", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		mockService := NewMockService(ctrl)
// 		mockTriton := NewMockTriton(ctrl)
// 		h := handler.NewHandler(mockService, mockTriton)

// 		mockTriton.
// 			EXPECT().
// 			IsTritonServerReady().
// 			Return(true)

// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
// 		defer cancel()
// 		liveRes, err := h.Liveness(ctx, &modelPB.LivenessRequest{})
// 		assert.NoError(t, err)
// 		assert.Equal(t, liveRes.HealthCheckResponse.Status, healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING)
// 	})
// }
