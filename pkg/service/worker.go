package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"go.temporal.io/api/enums/v1"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

func (s *service) GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, err
	}

	workflowExecutionInfo := workflowExecutionRes.WorkflowExecutionInfo

	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var result error
		workflowRun := s.temporalClient.GetWorkflow(ctx, workflowID, "")
		err = workflowRun.Get(ctx, &result)
		if err != nil {
			return nil, err
		}
		if result != nil {
			operation = longrunningpb.Operation{
				Done: true,
				Result: &longrunningpb.Operation_Error{
					Error: &status.Status{
						Code:    int32(enums.WORKFLOW_EXECUTION_STATUS_FAILED),
						Details: []*anypb.Any{},
						Message: result.Error(),
					},
				},
			}
		} else {
			operation = longrunningpb.Operation{
				Done: true,
				Result: &longrunningpb.Operation_Response{
					Response: &anypb.Any{},
				},
			}
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
				Error: &status.Status{
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
