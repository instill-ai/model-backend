package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/worker"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
	workflowpb "go.temporal.io/api/workflow/v1"
)

func (s *service) DeployModelInstanceAsync(owner string, modelUID uuid.UUID, modelInstanceUID uuid.UUID) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"DeployModelWorkflow",
		&worker.ModelInstanceParams{
			ModelUID:         modelUID,
			ModelInstanceUID: modelInstanceUID,
			Owner:            owner,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployModelInstanceAsync(owner string, modelUID uuid.UUID, modelInstanceUID uuid.UUID) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"UnDeployModelWorkflow",
		&worker.ModelInstanceParams{
			ModelUID:         modelUID,
			ModelInstanceUID: modelInstanceUID,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, *worker.ModelInstanceParams, string, error) {
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

	// Get search attributes that were provided when workflow was started.
	modelInstanceParams := worker.ModelInstanceParams{}
	operationType := ""
	for k, v := range workflowExecutionInfo.GetSearchAttributes().GetIndexedFields() {
		if k != "ModelUID" && k != "ModelInstanceUID" && k != "Owner" && k != "Type" {
			continue
		}
		var currentVal string
		if err := converter.GetDefaultDataConverter().FromPayload(v, &currentVal); err != nil {
			return nil, nil, "", err
		}
		if currentVal == "" {
			continue
		}

		if k == "Type" {
			operationType = currentVal
			continue
		}

		uid, err := uuid.FromString(currentVal)
		if err != nil {
			return nil, nil, "", err
		}
		if k == "ModelUID" {
			modelInstanceParams.ModelUID = uid
		} else if k == "ModelInstanceUID" {
			modelInstanceParams.ModelInstanceUID = uid
		} else if k == "Owner" {
			modelInstanceParams.Owner = fmt.Sprintf("users/%s", currentVal) // remove prefix users when storing in temporal
		}
	}
	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, &modelInstanceParams, operationType, nil
}

func (s *service) GetOperation(workflowId string) (*longrunningpb.Operation, *worker.ModelInstanceParams, string, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(context.Background(), workflowId, "")

	if err != nil {
		return nil, nil, "", err
	}
	return getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) ListOperation(pageSize int, pageToken string) ([]*longrunningpb.Operation, []*worker.ModelInstanceParams, string, int64, error) {
	var executions []*workflowpb.WorkflowExecutionInfo
	// could support query such as by model or model instance
	resp, err := s.temporalClient.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     modelWorker.Namespace,
		PageSize:      int32(pageSize),
		NextPageToken: []byte(pageToken),
	})
	if err != nil {
		return nil, nil, "", 0, err
	}

	executions = append(executions, resp.Executions...)
	var operations []*longrunningpb.Operation
	var modelInstanceParams []*worker.ModelInstanceParams
	for _, wf := range executions {
		operation, modelInstanceParam, _, err := getOperationFromWorkflowInfo(wf)

		if err != nil {
			return nil, nil, "", 0, err
		}
		operations = append(operations, operation)
		modelInstanceParams = append(modelInstanceParams, modelInstanceParam)
	}
	return operations, modelInstanceParams, string(resp.NextPageToken), int64(len(operations)), nil
}

func (s *service) CancelOperation(workflowId string) error {
	return s.temporalClient.CancelWorkflow(context.Background(), workflowId, "")
}

func (s *service) CreateModelAsync(owner string, model *datamodel.Model) (string, error) {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"CreateModelWorkflow",
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

func (s *service) SearchAttributeReady() error {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	ctx := context.Background()
	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"AddSearchAttributeWorkflow",
	)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	start := time.Now()
	for {
		if time.Since(start) > 10*time.Second {
			return fmt.Errorf("health workflow timed out")
		}
		workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, we.GetID(), we.GetRunID())
		if err != nil {
			continue
		}
		if workflowExecutionRes.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			return nil
		} else if workflowExecutionRes.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
			return fmt.Errorf("health workflow failed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
