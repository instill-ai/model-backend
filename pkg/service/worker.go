package service

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/api/enums/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/model-backend/pkg/resource"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	resourcex "github.com/instill-ai/x/resource"
)

func (s *service) GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	return s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo, workflowID)
}

func (s *service) GetNamespaceLatestModelOperation(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*longrunningpb.Operation, error) {
	ownerPermalink := ns.Permalink()

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)

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

	outputWorkflowID, err := s.redisClient.Get(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", userUID, requesterUID, dbModel.UID.String(), "")).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	workflowID, err := resource.GetWorkflowID(outputWorkflowID)
	if err != nil {
		return nil, err
	}

	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	operation, err := s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo, workflowID)
	if err != nil {
		return nil, err
	}

	if view != modelpb.View_VIEW_FULL {
		operation.Result = nil
	}

	return operation, nil

}

func (s *service) GetNamespaceModelOperation(ctx context.Context, ns resource.Namespace, modelID string, version string, view modelpb.View) (*longrunningpb.Operation, error) {
	ownerPermalink := ns.Permalink()

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)

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

	outputWorkflowID, err := s.redisClient.Get(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s:%s:%s", userUID, requesterUID, dbModel.UID.String(), version)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	workflowID, err := resource.GetWorkflowID(outputWorkflowID)
	if err != nil {
		return nil, err
	}

	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	operation, err := s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo, workflowID)
	if err != nil {
		return nil, err
	}

	if view != modelpb.View_VIEW_FULL {
		operation.Result = nil
	}

	return operation, nil

}

func (s *service) getOperationFromWorkflowInfo(ctx context.Context, workflowExecutionInfo *workflowpb.WorkflowExecutionInfo, triggerUID string) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:

		trigger, err := s.repository.GetModelRunByUID(ctx, triggerUID)
		if err != nil {
			return nil, err
		}

		_, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
		input, err := s.minioClient.GetFile(ctx, userUID, trigger.InputReferenceID)
		if err != nil {
			return nil, err
		}
		if !trigger.OutputReferenceID.Valid {
			return nil, fmt.Errorf("trigger output not valid")
		}
		output, err := s.minioClient.GetFile(ctx, userUID, trigger.OutputReferenceID.String)
		if err != nil {
			return nil, err
		}

		latestOperation := &modelpb.LatestOperation{}
		triggerModelReq := &modelpb.TriggerNamespaceModelRequest{}
		triggerModelResp := &modelpb.TriggerNamespaceModelResponse{}

		if err := protojson.Unmarshal(input, triggerModelReq); err != nil {
			return nil, err
		}

		if err := protojson.Unmarshal(output, triggerModelResp); err != nil {
			return nil, err
		}

		// TODO: handle mimetype for output
		// for i := range triggerModelResp.TaskOutputs {
		// 	ttiOutput := triggerModelResp.TaskOutputs[i].GetTextToImage()
		// 	if ttiOutput != nil {
		// 		for i := range ttiOutput.Images {
		// 			ttiOutput.Images[i] = fmt.Sprintf("data:image/jpeg;base64,%s", ttiOutput.Images[i])
		// 		}
		// 	}
		// }

		latestOperation.Request = triggerModelReq
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
		var s *structpb.Value

		errMessage, ok := workflowExecutionInfo.GetMemo().GetFields()["error"]
		if ok {
			s = structpb.NewStringValue(string(errMessage.GetData()))
		} else {
			s = structpb.NewStringValue("model execution error")
		}

		errMessagePB, err := anypb.New(s)
		if err != nil {
			return nil, err
		}

		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &rpcStatus.Status{
					Code:    13,
					Details: []*anypb.Any{errMessagePB},
					Message: s.GetStringValue(),
				},
			},
		}
	}

	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, nil
}
