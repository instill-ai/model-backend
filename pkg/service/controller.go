package service

import (
	"context"

	"github.com/instill-ai/model-backend/pkg/util"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func (s *service) GetResourceState(ctx context.Context, modelID string) (*modelPB.Model_State, error) {
	resourceName := util.ConvertModelToResourceName(modelID)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetModelState().Enum(), nil
}

func (s *service) UpdateResourceState(ctx context.Context, modelID string, state modelPB.Model_State, progress *int32, workflowId *string) error {
	resourceName := util.ConvertModelToResourceName(modelID)

	if _, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ModelState{
				ModelState: state,
			},
			Progress: progress,
		},
		WorkflowId: workflowId,
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(ctx context.Context, modelID string) error {
	resourceName := util.ConvertModelToResourceName(modelID)

	if _, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: resourceName,
	}); err != nil {
		return err
	}

	return nil
}
