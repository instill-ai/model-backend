package service

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
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

func (s *service) GetRscNamespace(ctx context.Context, namespaceID string) (resource.Namespace, error) {

	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceAdmin(ctx, &mgmtpb.CheckNamespaceAdminRequest{
		Id: namespaceID,
	})
	if err != nil {
		return resource.Namespace{}, err
	}
	if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER {
		return resource.Namespace{
			NsType: resource.User,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	} else if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION {
		return resource.Namespace{
			NsType: resource.Organization,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	}
	return resource.Namespace{}, fmt.Errorf("namespace error")
}

func (s *service) pageSizeInRange(pageSize int32) int32 {
	if pageSize <= 0 {
		return repository.DefaultPageSize
	}

	if pageSize > repository.MaxPageSize {
		return repository.MaxPageSize
	}

	return pageSize
}

func (s *service) pageInRange(page int32) int32 {
	if page <= 0 {
		return 0
	}

	return page
}

// CanViewPrivateData - only with credit owner ns could users see their input/output data
func CanViewPrivateData(namespace, requesterUID string) bool {
	return namespace == requesterUID
}
