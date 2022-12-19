package service

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/worker"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	modelWorker "github.com/instill-ai/model-backend/internal/worker"
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

func getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunning.Operation, *worker.ModelInstanceParams, error) {
	operation := longrunning.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		operation = longrunning.Operation{
			Done: true,
			Result: &longrunning.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		operation = longrunning.Operation{
			Done: false,
			Result: &longrunning.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	default:
		operation = longrunning.Operation{
			Done: true,
			Result: &longrunning.Operation_Error{
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
	for k, v := range workflowExecutionInfo.GetSearchAttributes().GetIndexedFields() {
		if k != "ModelUID" && k != "ModelInstanceUID" && k != "Owner" {
			continue
		}
		var currentVal string
		if err := converter.GetDefaultDataConverter().FromPayload(v, &currentVal); err != nil {
			return nil, nil, err
		}
		if currentVal == "" {
			continue
		}
		uid, err := uuid.FromString(currentVal)
		if err != nil {
			return nil, nil, err
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
	return &operation, &modelInstanceParams, nil
}

func (s *service) GetOperation(workflowId string) (*longrunning.Operation, *worker.ModelInstanceParams, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(context.Background(), workflowId, "")
	if err != nil {
		return nil, nil, err
	}

	return getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) ListOperation(pageSize int, pageToken string) ([]*longrunning.Operation, []*worker.ModelInstanceParams, string, int64, error) {
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
	var operations []*longrunning.Operation
	var modelInstanceParams []*worker.ModelInstanceParams
	for _, wf := range executions {
		operation, modelInstanceParam, err := getOperationFromWorkflowInfo(wf)

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
