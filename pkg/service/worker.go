package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/worker"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (s *service) DeployUserModelAsync(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, modelUID uuid.UUID) (string, error) {

	logger, _ := logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	model, err := s.repository.GetUserModelByUID(ctx, ownerPermalink, userPermalink, modelUID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"DeployModelWorkflow",
		&worker.ModelParams{
			Model:          model,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployUserModelAsync(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, modelUID uuid.UUID) (string, error) {

	logger, _ := logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	model, err := s.repository.GetUserModelByUID(ctx, ownerPermalink, userPermalink, modelUID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"UnDeployModelWorkflow",
		&worker.ModelParams{
			Model:          model,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowId, "")
	if err != nil {
		return nil, err
	}

	workflowExecutionInfo := workflowExecutionRes.WorkflowExecutionInfo

	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var result error
		workflowRun := s.temporalClient.GetWorkflow(ctx, workflowId, "")
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

func (s *service) CreateUserModelAsync(ctx context.Context, model *datamodel.Model) (string, error) {

	logger, _ := logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"CreateModelWorkflow",
		&worker.ModelParams{
			Model:          model,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) DeployUserModelAsyncAdmin(ctx context.Context, modelUID uuid.UUID) (string, error) {

	logger, _ := logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	model, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"DeployModelWorkflow",
		&worker.ModelParams{
			Model:          model,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployUserModelAsyncAdmin(ctx context.Context, userUID uuid.UUID, modelUID uuid.UUID) (string, error) {

	logger, _ := logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	model, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"UnDeployModelWorkflow",
		&worker.ModelParams{
			Model:          model,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}
