package service

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
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
	switch resp.Type {
	case mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER:
		return resource.Namespace{
			NsType: resource.User,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	case mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION:
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

func parseMetadataToStructArr(metadataMap map[string][]byte, run *datamodel.ModelRun) ([]*structpb.Struct, []*structpb.Struct, error) {
	data, ok := metadataMap[run.InputReferenceID]
	if !ok {
		return nil, nil, fmt.Errorf("key doesn't exist")
	}
	// todo: fix TaskInputs type
	triggerReq := &modelpb.TriggerNamespaceModelRequest{}
	err := protojson.Unmarshal(data, triggerReq)
	if err != nil {
		return nil, nil, err
	}

	var taskOutputs []*structpb.Struct
	if run.OutputReferenceID.Valid {
		data, ok = metadataMap[run.OutputReferenceID.String]
		if !ok {
			return triggerReq.TaskInputs, nil, fmt.Errorf("key doesn't exist")
		}

		// todo: fix TaskOutputs type
		triggerModelResp := &modelpb.TriggerNamespaceModelResponse{}
		err = protojson.Unmarshal(data, triggerModelResp)
		if err != nil {
			return triggerReq.TaskInputs, nil, err
		}

		taskOutputs = triggerModelResp.TaskOutputs
	}
	return triggerReq.TaskInputs, taskOutputs, nil
}

func convertModelRunToPB(run *datamodel.ModelRun) *modelpb.ModelRun {
	pbModelRun := &modelpb.ModelRun{
		Uid:              run.UID.String(),
		ModelId:          &run.Model.ID,
		ModelNamespaceId: run.Model.NamespaceID,
		Version:          run.ModelVersion,
		Status:           runpb.RunStatus(run.Status),
		Source:           runpb.RunSource(run.Source),
		Error:            run.Error.Ptr(),
		CreateTime:       timestamppb.New(run.CreateTime),
		UpdateTime:       timestamppb.New(run.UpdateTime),
	}

	if run.TotalDuration.Valid {
		totalDuration := int32(run.TotalDuration.Int64)
		pbModelRun.TotalDuration = &totalDuration
	}
	if run.EndTime.Valid {
		pbModelRun.EndTime = timestamppb.New(run.EndTime.Time)
	}

	return pbModelRun
}
