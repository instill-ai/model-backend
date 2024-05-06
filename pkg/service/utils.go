package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
)

func (s *service) checkNamespacePermission(ctx context.Context, ns resource.Namespace) error {
	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return ErrNoPermission
		}
	} else if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
		return ErrNoPermission
	}
	return nil
}
