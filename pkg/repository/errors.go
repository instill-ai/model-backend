package repository

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrPageTokenDecode = status.New(codes.InvalidArgument, "page token decode error").Err()
var ErrOwnerTypeNotMatch = status.New(codes.InvalidArgument, "owner type not match").Err()
var ErrNoDataDeleted = status.New(codes.NotFound, "no data deleted").Err()
var ErrNoDataUpdated = status.New(codes.NotFound, "no data updated").Err()
