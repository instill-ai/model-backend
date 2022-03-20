package rpc

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/gogo/status"
	gomock "github.com/golang/mock/gomock"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	model "github.com/instill-ai/protogen-go/model/v1alpha"
	"github.com/stretchr/testify/assert"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const NAMESPACE = "local-user"

// RecoveryInterceptor - panic handler
func recoveryInterceptorOpt() grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	})
}

// CustomInterceptor - append metadatas for unary
func unaryAppendMetadataInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "can not extract metadata")
	}

	md.Append("Username", "local-user")

	newCtx := metadata.NewIncomingContext(ctx, md)

	h, err := handler(newCtx, req)

	return h, err
}

// CustomInterceptor - append metadatas for stream
func streamAppendMetadataInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Internal, "can not extract metadata")
	}

	md.Append("Username", "local-user")

	newCtx := metadata.NewIncomingContext(stream.Context(), md)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx

	err := handler(srv, wrapped)

	return err
}

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

func dialer(t *testing.T) func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	ctrl := gomock.NewController(t)

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamAppendMetadataInterceptor,
			grpc_recovery.StreamServerInterceptor(recoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAppendMetadataInterceptor,
			grpc_recovery.UnaryServerInterceptor(recoveryInterceptorOpt()),
		)),
	}

	server := grpc.NewServer(grpcServerOpts...)

	mockModelService := NewMockModelService(ctrl)
	mockTritonService := NewMockTritonService(ctrl)
	modelServiceHandler := NewServiceHandlers(mockModelService, mockTritonService)

	model.RegisterModelServiceServer(server, modelServiceHandler)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

// func Test_CreateModelBinaryFileUpload(t *testing.T) {
// 	t.Run("CreateModelBinaryFileUpload", func(t *testing.T) {

// 		ctx := context.Background()
// 		conn, _ := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer(t)))
// 		defer conn.Close()

// 		c := model.NewModelServiceClient(conn)

// 		modelName := "test"
// 		description := "test model"
// 		streamUploader, _ := c.CreateModelBinaryFileUpload(ctx)

// 		defer streamUploader.CloseSend()

// 		const chunkSize = 64 * 1024 // 64 KiB
// 		buf := make([]byte, chunkSize)
// 		firstChunk := true

// 		pwd, _ := os.Getwd()
// 		file, _ := os.Open(pwd + "/../data/dummy-det-model.zip")

// 		defer file.Close()

// 		for {
// 			n, errRead := file.Read(buf)
// 			if errRead != nil {
// 				if errRead == io.EOF {
// 					break
// 				}

// 				break
// 			}
// 			if firstChunk {
// 				_ = streamUploader.Send(&model.CreateModelBinaryFileUploadRequest{
// 					ModelInitData: &model.ModelInitData{
// 						Name:        modelName,
// 						Description: description,
// 						Byte:        buf[:n],
// 					},
// 				})
// 				firstChunk = false
// 			} else {
// 				_ = streamUploader.Send(&model.CreateModelBinaryFileUploadRequest{
// 					ModelInitData: &model.ModelInitData{
// 						Byte: buf[:n],
// 					},
// 				})
// 			}
// 		}

// 		response, err := streamUploader.CloseAndRecv()
// 		fmt.Println(">>>>> res ", response, err)
// 		assert.NoError(t, err)
// 	})
// }
