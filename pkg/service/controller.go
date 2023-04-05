package service

import (
	"context"
	"time"

	"github.com/instill-ai/model-backend/pkg/util"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func (s *service) GetResourceState(modelID string) (*modelPB.Model_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertModelToResourceName(modelID)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetModelState().Enum(), nil
}

func (s *service) UpdateResourceState(modelID string, state modelPB.Model_State, progress *int32, workflowId *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertModelToResourceName(modelID)

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ModelState{
				ModelState: state,
			},
			Progress: progress,
		},
		WorkflowId: workflowId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(modelID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertModelToResourceName(modelID)

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return err
	}

	return nil
}
