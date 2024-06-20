package handler

import (
	"context"
	"errors"
	"fmt"

	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/x/sterr"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (h *PrivateHandler) ListModelsAdmin(ctx context.Context, req *modelpb.ListModelsAdminRequest) (*modelpb.ListModelsAdminResponse, error) {

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListModelsAdmin(ctx, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
	if err != nil {
		return &modelpb.ListModelsAdminResponse{}, err
	}

	resp := modelpb.ListModelsAdminResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpModelAdmin(ctx context.Context, req *modelpb.LookUpModelAdminRequest) (*modelpb.LookUpModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		return &modelpb.LookUpModelAdminResponse{}, err
	}

	pbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, parseView(req.GetView()))
	if err != nil {
		return &modelpb.LookUpModelAdminResponse{}, err
	}

	return &modelpb.LookUpModelAdminResponse{Model: pbModel}, nil
}

type DeployNamespaceModelAdminRequestInterface interface {
	GetName() string
	GetVersion() string
	GetDigest() string
}

func (h *PrivateHandler) DeployUserModelAdmin(ctx context.Context, req *modelpb.DeployUserModelAdminRequest) (resp *modelpb.DeployUserModelAdminResponse, err error) {

	err = h.deployNamespaceModelAdmin(ctx, req)

	return &modelpb.DeployUserModelAdminResponse{}, err
}

func (h *PrivateHandler) DeployOrganizationModelAdmin(ctx context.Context, req *modelpb.DeployOrganizationModelAdminRequest) (resp *modelpb.DeployOrganizationModelAdminResponse, err error) {

	err = h.deployNamespaceModelAdmin(ctx, req)

	return &modelpb.DeployOrganizationModelAdminResponse{}, err
}

func (h *PrivateHandler) deployNamespaceModelAdmin(ctx context.Context, req DeployNamespaceModelAdminRequestInterface) error {

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		return err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return err
	}

	version := &datamodel.ModelVersion{
		Name:     req.GetName(),
		Version:  req.GetVersion(),
		Digest:   req.GetDigest(),
		ModelUID: uuid.FromStringOrNil(pbModel.Uid),
	}

	if err := h.service.CreateModelVersionAdmin(ctx, version); err != nil {
		return err
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
			return errors.New(e.Error())
		}
		return st.Err()
	}

	return nil
}

type UndeployNamespaceModelAdminRequestInterface interface {
	GetName() string
	GetVersion() string
	GetDigest() string
}

func (h *PrivateHandler) UndeployUserModelAdmin(ctx context.Context, req *modelpb.UndeployUserModelAdminRequest) (resp *modelpb.UndeployUserModelAdminResponse, err error) {

	err = h.undeployNamespaceModelAdmin(ctx, req)

	return &modelpb.UndeployUserModelAdminResponse{}, err
}

func (h *PrivateHandler) UndeployOrganizationModelAdmin(ctx context.Context, req *modelpb.UndeployOrganizationModelAdminRequest) (resp *modelpb.UndeployOrganizationModelAdminResponse, err error) {

	err = h.undeployNamespaceModelAdmin(ctx, req)

	return &modelpb.UndeployOrganizationModelAdminResponse{}, err
}

func (h *PrivateHandler) undeployNamespaceModelAdmin(ctx context.Context, req UndeployNamespaceModelAdminRequestInterface) error {

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		return err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return err
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
			return errors.New(e.Error())
		}
		return st.Err()
	}

	if err := h.service.DeleteModelVersionAdmin(ctx, uuid.FromStringOrNil(pbModel.Uid), version.Version); err != nil {
		return err
	}
	return nil
}
