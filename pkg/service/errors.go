package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNoPermission = status.New(codes.PermissionDenied, "The caller does not have permission to execute the specified operation").Err()
var ErrNotFound = status.New(codes.NotFound, "Some requested entity (e.g., Model namespace, Model instance) was not found").Err()
var ErrUnauthenticated = status.New(codes.Unauthenticated, "The request does not have valid authentication credentials for the operation").Err()

var ErrRateLimiting = status.New(codes.FailedPrecondition, "rate limiting").Err()
var ErrExceedMaxBatchSize = status.New(codes.InvalidArgument, "the batch size can not exceed 32").Err()
