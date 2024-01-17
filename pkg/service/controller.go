package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/utils"
	controllerPB "github.com/instill-ai/protogen-go/model/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (s *service) GetResourceState(ctx context.Context, modelUID uuid.UUID) (*modelPB.Model_State, error) {
	resourcePermalink := utils.ConvertModelToResourcePermalink(modelUID.String())

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetModelState().Enum(), nil
}

func (s *service) UpdateResourceState(ctx context.Context, modelUID uuid.UUID, state modelPB.Model_State, progress *int32, workflowID *string) error {
	resourcePermalink := utils.ConvertModelToResourcePermalink(modelUID.String())

	if _, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State: &controllerPB.Resource_ModelState{
				ModelState: state,
			},
			Progress: progress,
		},
		WorkflowId: workflowID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(ctx context.Context, modelUID uuid.UUID) error {
	resourcePermalink := utils.ConvertModelToResourcePermalink(modelUID.String())

	if _, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		ResourcePermalink: resourcePermalink,
	}); err != nil {
		return err
	}

	return nil
}
