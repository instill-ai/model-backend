package handler

import (
	"context"
	"fmt"
	"strings"

	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/resource"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
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

func (h *PrivateHandler) DeployUserModelAdmin(ctx context.Context, req *modelpb.DeployUserModelAdminRequest) (*modelpb.DeployUserModelAdminResponse, error) {
	if _, err := h.DeployNamespaceModelAdmin(ctx, &modelpb.DeployNamespaceModelAdminRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.GetVersion(),
		Digest:      req.GetDigest(),
	}); err != nil {
		return nil, err
	}

	return &modelpb.DeployUserModelAdminResponse{}, nil
}

func (h *PrivateHandler) DeployOrganizationModelAdmin(ctx context.Context, req *modelpb.DeployOrganizationModelAdminRequest) (*modelpb.DeployOrganizationModelAdminResponse, error) {
	if _, err := h.DeployNamespaceModelAdmin(ctx, &modelpb.DeployNamespaceModelAdminRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.GetVersion(),
		Digest:      req.GetDigest(),
	}); err != nil {
		return nil, err
	}

	return &modelpb.DeployOrganizationModelAdminResponse{}, nil
}

func (h *PrivateHandler) DeployNamespaceModelAdmin(ctx context.Context, req *modelpb.DeployNamespaceModelAdminRequest) (*modelpb.DeployNamespaceModelAdminResponse, error) {

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, req.GetModelId(), modelpb.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	version := &datamodel.ModelVersion{
		Name:     ns.Name(),
		Version:  req.GetVersion(),
		Digest:   req.GetDigest(),
		ModelUID: uuid.FromStringOrNil(pbModel.Uid),
	}

	if _, err := h.service.GetModelVersionAdmin(ctx, uuid.FromStringOrNil(pbModel.Uid), version.Version); err != nil {
		if err := h.service.CreateModelVersionAdmin(ctx, version); err != nil {
			return nil, err
		}
	}

	if _, err := h.service.GetArtifactPrivateServiceClient().GetRepositoryTag(ctx, &artifactpb.GetRepositoryTagRequest{
		Name: fmt.Sprintf("repositories/%s/%s/tags/%s", ns.NsID, req.GetModelId(), version.Version),
	}); err != nil {
		return nil, err
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, req.GetModelId(), pbModel.GetHardware(), req.GetVersion(), ray.Deploy); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to deploy the model: %s", err.Error()))
	}

	return &modelpb.DeployNamespaceModelAdminResponse{}, nil
}

type UndeployNamespaceModelAdminRequestInterface interface {
	GetName() string
	GetVersion() string
	GetDigest() string
}

func (h *PrivateHandler) UndeployUserModelAdmin(ctx context.Context, req *modelpb.UndeployUserModelAdminRequest) (*modelpb.UndeployUserModelAdminResponse, error) {
	if _, err := h.UndeployNamespaceModelAdmin(ctx, &modelpb.UndeployNamespaceModelAdminRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.GetVersion(),
		Digest:      req.GetDigest(),
	}); err != nil {
		return nil, err
	}

	return &modelpb.UndeployUserModelAdminResponse{}, nil
}

func (h *PrivateHandler) UndeployOrganizationModelAdmin(ctx context.Context, req *modelpb.UndeployOrganizationModelAdminRequest) (*modelpb.UndeployOrganizationModelAdminResponse, error) {
	if _, err := h.UndeployNamespaceModelAdmin(ctx, &modelpb.UndeployNamespaceModelAdminRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.GetVersion(),
		Digest:      req.GetDigest(),
	}); err != nil {
		return nil, err
	}

	return &modelpb.UndeployOrganizationModelAdminResponse{}, nil
}

func (h *PrivateHandler) UndeployNamespaceModelAdmin(ctx context.Context, req *modelpb.UndeployNamespaceModelAdminRequest) (*modelpb.UndeployNamespaceModelAdminResponse, error) {

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, req.GetModelId(), modelpb.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, req.GetModelId(), pbModel.GetHardware(), req.GetVersion(), ray.Undeploy); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to undeploy the model: %s", err.Error()))
	}

	return &modelpb.UndeployNamespaceModelAdminResponse{}, nil
}
