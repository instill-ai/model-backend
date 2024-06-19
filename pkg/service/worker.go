package service

import (
	"context"
	"errors"
	"fmt"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"go.temporal.io/api/enums/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/go-redis/redis/v9"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (s *service) GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	return s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo, nil)
}

func (s *service) GetNamespaceLatestModelOperation(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*longrunningpb.Operation, error) {
	ownerPermalink := ns.Permalink()

	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	triggerModelReq := &modelpb.TriggerUserModelRequest{}

	inputJSON, err := s.redisClient.Get(ctx, fmt.Sprintf("model_trigger_input:%s:%s", userUID, dbModel.UID.String())).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	err = protojson.Unmarshal(inputJSON, triggerModelReq)
	if err != nil {
		return nil, err
	}

	outputWorkflowID := s.redisClient.Get(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s", userUID, dbModel.UID.String())).Val()
	operationID, err := resource.GetOperationID(outputWorkflowID)
	if err != nil {
		return nil, err
	}

	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, operationID, "")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	operation, err := s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo, triggerModelReq)
	if err != nil {
		return nil, err
	}

	if view != modelpb.View_VIEW_FULL {
		operation.Result = nil
	}

	return operation, nil

}

func (s *service) getOperationFromWorkflowInfo(ctx context.Context, workflowExecutionInfo *workflowpb.WorkflowExecutionInfo, triggerModelReq *modelpb.TriggerUserModelRequest) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:

		latestOperation := &modelpb.LatestOperation{
			Request: triggerModelReq,
		}

		triggerModelResp := &modelpb.TriggerUserModelResponse{}

		blobRedisKey := fmt.Sprintf("async_model_response:%s", workflowExecutionInfo.Execution.WorkflowId)
		blob, err := s.redisClient.Get(ctx, blobRedisKey).Bytes()
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(blob, triggerModelResp)
		if err != nil {
			return nil, err
		}

		// TODO: handle mimetype for output
		for i := range triggerModelResp.TaskOutputs {
			ttiOutput := triggerModelResp.TaskOutputs[i].GetTextToImage()
			if ttiOutput != nil {
				for i := range ttiOutput.Images {
					ttiOutput.Images[i] = fmt.Sprintf("data:image/jpeg;base64,%s", ttiOutput.Images[i])
				}
			}
		}

		latestOperation.Response = triggerModelResp

		resp, err := anypb.New(latestOperation)
		if err != nil {
			return nil, err
		}
		resp.TypeUrl = "buf.build/instill-ai/protobufs/model.model.v1alpha.LatestOperation"
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: resp,
			},
		}
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		operation = longrunningpb.Operation{
			Done: false,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	default:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &rpcStatus.Status{
					Code:    int32(workflowExecutionInfo.Status),
					Details: []*anypb.Any{},
					Message: "",
				},
			},
		}
	}

	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, nil
}
