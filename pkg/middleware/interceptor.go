package middleware

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/repository"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

// RecoveryInterceptor - panic handler
func RecoveryInterceptorOpt() grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	})
}

// CustomInterceptor - append metadatas for unary
func UnaryAppendMetadataInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "can not extract metadata")
	}

	// TODO: Replace with decoded JWT header
	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient()
	defer mgmtPrivateServiceClientConn.Close()
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	userResp, err := mgmtPrivateServiceClient.ListUsersAdmin(context.Background(), &mgmtPB.ListUsersAdminRequest{
		PageSize:  &userPageSizeMax,
		PageToken: &userPageToken,
	})
	if err == nil && len(userResp.Users) > 0 && userResp.Users[0].GetUid() != "" {
		md.Append("owner", fmt.Sprintf("users/%s", userResp.Users[0].GetUid()))
	} else {
		md.Append("owner", "users/45d19b6d-5073-4bc7-b3c6-b668ea98b3c4")
	}

	newCtx := metadata.NewIncomingContext(ctx, md)

	h, err := handler(newCtx, req)

	return h, err
}

// CustomInterceptor - append metadatas for stream
func StreamAppendMetadataInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Internal, "can not extract metadata")
	}

	// TODO: Replace with decoded JWT header
	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient()
	defer mgmtPrivateServiceClientConn.Close()
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	userResp, er := mgmtPrivateServiceClient.ListUsersAdmin(context.Background(), &mgmtPB.ListUsersAdminRequest{
		PageSize:  &userPageSizeMax,
		PageToken: &userPageToken,
	})
	if er == nil && len(userResp.Users) > 0 && userResp.Users[0].GetUid() != "" {
		md.Append("owner", fmt.Sprintf("users/%s", userResp.Users[0].GetUid()))
	} else {
		md.Append("owner", "users/45d19b6d-5073-4bc7-b3c6-b668ea98b3c4")
	}

	newCtx := metadata.NewIncomingContext(stream.Context(), md)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx

	err := handler(srv, wrapped)

	return err
}
