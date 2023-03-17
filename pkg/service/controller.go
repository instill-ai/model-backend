package service

import (
	"context"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

func (s *service) GetResourceState(name string) (*datamodel.ResourceState, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: name,
	})

	if err != nil {
		return nil, err
	}

	state := datamodel.ResourceState{
		Name: resp.Resource.Name,
		State: resp.Resource.State,
		Progress: resp.Resource.Progress,
	}

	return &state, nil
}

func (s *service) UpdateResourceState(state *datamodel.ResourceState, workflowId string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: state.Name,
			State: state.State,
			Progress: state.Progress,
		},
		WorkflowId: &workflowId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(name string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: name,
	})

	if err != nil {
		return err
	}

	return nil
}