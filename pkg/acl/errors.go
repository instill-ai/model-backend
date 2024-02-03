package acl

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrMembershipNotFound = status.New(codes.NotFound, "membership not found").Err()
