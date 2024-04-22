package handler

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/x/sterr"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (h *PrivateHandler) ListModelsAdmin(ctx context.Context, req *modelPB.ListModelsAdminRequest) (*modelPB.ListModelsAdminResponse, error) {

	pbModels, totalSize, nextPageToken, err := h.service.ListModelsAdmin(ctx, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), req.GetShowDeleted())
	if err != nil {
		return &modelPB.ListModelsAdminResponse{}, err
	}

	resp := modelPB.ListModelsAdminResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpModelAdmin(ctx context.Context, req *modelPB.LookUpModelAdminRequest) (*modelPB.LookUpModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	pbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, parseView(req.GetView()))
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	return &modelPB.LookUpModelAdminResponse{Model: pbModel}, nil
}

func (h *PrivateHandler) DeployModelAdmin(ctx context.Context, req *modelPB.DeployModelAdminRequest) (*modelPB.DeployModelAdminResponse, error) {

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	version := &datamodel.ModelVersion{
		Name:     req.GetName(),
		Version:  req.GetVersion(),
		Digest:   req.GetDigest(),
		ModelUID: uuid.FromStringOrNil(pbModel.Uid),
	}

	if err := h.service.CreateModelVersionAdmin(ctx, version); err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, modelID, pbModel.GetHardware(), req.GetVersion(), true); err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] deploy a model error: %s", err.Error()),
			"ray-server",
			"deploy model",
			"",
			err.Error(),
		)

		if e != nil {
			return &modelPB.DeployModelAdminResponse{}, errors.New(e.Error())
		}
		return &modelPB.DeployModelAdminResponse{}, st.Err()
	}

	return &modelPB.DeployModelAdminResponse{}, nil
}

func (h *PrivateHandler) UndeployModelAdmin(ctx context.Context, req *modelPB.UndeployModelAdminRequest) (*modelPB.UndeployModelAdminResponse, error) {

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	version := &datamodel.ModelVersion{
		Name:     req.GetName(),
		Version:  req.GetVersion(),
		Digest:   req.GetDigest(),
		ModelUID: uuid.FromStringOrNil(pbModel.Uid),
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, modelID, pbModel.GetHardware(), req.GetVersion(), false); err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] undeploy a model error: %s", err.Error()),
			"ray-server",
			"undeploy model",
			"",
			err.Error(),
		)

		if e != nil {
			return &modelPB.UndeployModelAdminResponse{}, errors.New(e.Error())
		}
		return &modelPB.UndeployModelAdminResponse{}, st.Err()
	}

	if err := h.service.DeleteModelVersionAdmin(ctx, version); err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}
	return &modelPB.UndeployModelAdminResponse{}, nil
}
