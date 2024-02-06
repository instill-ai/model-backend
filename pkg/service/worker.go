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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"

	grpc_status "google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/worker"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
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

func (s *service) CreateNamespaceModelAsync(ctx context.Context, ns resource.Namespace, authUser *AuthUser, model *datamodel.Model) (string, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, constant.HeaderUserUIDKey, resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("organizations/%s", ns.NsID)})
		if err != nil {
			s, ok := grpc_status.FromError(err)
			if !ok {
				return "", err
			}
			if s.Code() != codes.Unimplemented {
				return "", err
			}
		} else if resp.Subscription.Plan == "inactive" {
			return "", grpc_status.Errorf(codes.FailedPrecondition, "the organization subscription is not active")
		}

		granted, err := s.aclClient.CheckPermission("organization", ns.NsUID, authUser.GetACLType(), authUser.UID, "member")
		if err != nil {
			return "", err
		}
		if !granted {
			return "", ErrNoPermission
		}
	} else if ns.NsUID != authUser.UID {
		return "", ErrNoPermission
	}

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
			Model: model,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) DeployNamespaceModelAsyncAdmin(ctx context.Context, modelUID uuid.UUID) (string, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	model, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, false)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"DeployModelWorkflow",
		&worker.ModelParams{
			Model: model,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployNamespaceModelAsyncAdmin(ctx context.Context, modelUID uuid.UUID) (string, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	model, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, true)
	if err != nil {
		return "", err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"UnDeployModelWorkflow",
		&worker.ModelParams{
			Model: model,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}
