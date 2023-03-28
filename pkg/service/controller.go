package service

import (
	"context"
	"time"

	"github.com/instill-ai/model-backend/pkg/util"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func (s *service) GetResourceState(modelID string, modelInstanceID string) (*datamodel.ResourceState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertResourceName(modelID, modelInstanceID)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return nil, err
	}

	state := datamodel.ResourceState{
		Name:     resp.Resource.Name,
		State:    resp.Resource.GetModelInstanceState(),
		Progress: resp.Resource.Progress,
	}

	return &state, nil
}

func (s *service) UpdateResourceState(modelID string, modelInstanceID string, state modelPB.ModelInstance_State, progress *int32, workflowId *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertResourceName(modelID, modelInstanceID)

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ModelInstanceState{
				ModelInstanceState: state,
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

func (s *service) DeleteResourceState(modelID string, modelInstanceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := util.ConvertResourceName(modelID, modelInstanceID)

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return err
	}

	return nil
}
