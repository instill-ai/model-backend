package handler

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrCheckUpdateImmutableFields = status.New(codes.InvalidArgument, "update immutable fields error").Err()
var ErrCheckOutputOnlyFields = status.New(codes.InvalidArgument, "can not contain output only fields").Err()
var ErrCheckRequiredFields = status.New(codes.InvalidArgument, "required fields missing").Err()
var ErrFieldMask = status.New(codes.InvalidArgument, "field mask error").Err()
var ErrResourceID = status.New(codes.InvalidArgument, "resource ID error").Err()
var ErrSematicVersion = status.New(codes.InvalidArgument, "not a legal version, should be the format vX.Y.Z or vX.Y.Z-identifiers").Err()
var ErrUpdateMask = status.New(codes.InvalidArgument, "update mask error").Err()
var ErrConnectorNamespace = status.New(codes.InvalidArgument, "can not use other's connector").Err()
