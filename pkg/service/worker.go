package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/worker"
	modelv1alpha "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	workflowpb "go.temporal.io/api/workflow/v1"
)

func (s *service) DeployModelAsync(ctx context.Context, owner string, modelUID uuid.UUID) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	model, err := s.repository.GetModelByUid(owner, modelUID, modelv1alpha.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"DeployModelWorkflow",
		&worker.ModelParams{
			Model: model,
			Owner: owner,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployModelAsync(ctx context.Context, owner string, modelUID uuid.UUID) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	model, err := s.repository.GetModelByUid(owner, modelUID, modelv1alpha.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"UnDeployModelWorkflow",
		&worker.ModelParams{
			Model: model,
			Owner: owner,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
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

func (s *service) GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowId, "")

	if err != nil {
		return nil, err
	}
	return getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) CreateModelAsync(ctx context.Context, owner string, model *datamodel.Model) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"CreateModelWorkflow",
		&worker.ModelParams{
			Model: *model,
			Owner: owner,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}
