package handler

import (
	"context"

	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/internal/resource"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

type PrivateHandler struct {
	modelPB.UnimplementedModelPrivateServiceServer
	service service.Service
	triton  triton.Triton
}

func NewPrivateHandler(s service.Service, t triton.Triton) modelPB.ModelPrivateServiceServer {
	datamodel.InitJSONSchema()
	return &PrivateHandler{
		service: s,
		triton:  t,
	}
}

func (h *PrivateHandler) CheckModelInstance(ctx context.Context, req *modelPB.CheckModelInstanceRequest) (*modelPB.CheckModelInstanceResponse, error) {
	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &modelPB.CheckModelInstanceResponse{}, err
	}

	modelID, instanceID, err := resource.GetModelInstanceID(req.Name)
	if err != nil {
		return &modelPB.CheckModelInstanceResponse{}, err
	}

	dbModel, err := h.service.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.CheckModelInstanceResponse{}, err
	}

	dbModelInstance, err := h.service.GetModelInstance(dbModel.UID, instanceID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.CheckModelInstanceResponse{}, err
	}

	state, err := h.service.CheckModel(dbModelInstance.UID)
	if err != nil {
		return &modelPB.CheckModelInstanceResponse{}, err
	}

	return &modelPB.CheckModelInstanceResponse{
		Resource: &controllerPB.Resource{
			Name: req.Name,
			State: *state,
			Progress: 0,
		},
	}, err
}
