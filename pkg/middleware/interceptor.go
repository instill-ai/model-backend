package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
)

// RecoveryInterceptor - panic handler
func RecoveryInterceptorOpt() grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandler(func(p any) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	})
}

// CustomInterceptor - append metadatas for unary
func UnaryAppendMetadataInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "can not extract metadata")
	}

	newCtx := metadata.NewIncomingContext(ctx, md)
	h, err := handler(newCtx, req)

	return h, err
}

// CustomInterceptor - append metadatas for stream
func StreamAppendMetadataInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Internal, "can not extract metadata")
	}

	newCtx := metadata.NewIncomingContext(stream.Context(), md)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx

	err := handler(srv, wrapped)

	return err
}
