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

func (s *service) DeployModelAsync(owner string, modelUID uuid.UUID) (string, error) {
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
		&worker.ModelParams{
			ModelUID: modelUID,
			Owner:    owner,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func (s *service) UndeployModelAsync(owner string, modelUID uuid.UUID) (string, error) {
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
		&worker.ModelParams{
			ModelUID: modelUID,
		})

	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return id.String(), nil
}

func getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, *worker.ModelParams, string, error) {
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
	modelParams := worker.ModelParams{}
	operationType := ""
	for k, v := range workflowExecutionInfo.GetSearchAttributes().GetIndexedFields() {
		if k != "ModelUID" && k != "Owner" && k != "Type" {
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
			modelParams.ModelUID = uid
		} else if k == "Owner" {
			modelParams.Owner = fmt.Sprintf("users/%s", currentVal) // remove prefix users when storing in temporal
		}
	}
	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, &modelParams, operationType, nil
}

func (s *service) GetOperation(workflowId string) (*longrunningpb.Operation, *worker.ModelParams, string, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(context.Background(), workflowId, "")

	if err != nil {
		return nil, nil, "", err
	}
	return getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) ListOperation(pageSize int, pageToken string) ([]*longrunningpb.Operation, []*worker.ModelParams, string, int64, error) {
	var executions []*workflowpb.WorkflowExecutionInfo
	// could support query such as by model uid
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
	var modelParams []*worker.ModelParams
	for _, wf := range executions {
		operation, modelParam, _, err := getOperationFromWorkflowInfo(wf)

		if err != nil {
			return nil, nil, "", 0, err
		}
		operations = append(operations, operation)
		modelParams = append(modelParams, modelParam)
	}
	return operations, modelParams, string(resp.NextPageToken), int64(len(operations)), nil
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
			ModelUID: model.UID,
			Owner:    owner,
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
