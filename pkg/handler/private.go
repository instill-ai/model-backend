package handler

import (
	"context"
	"fmt"
	"strings"

	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/resource"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

// parseAdminModelVersionName parses a model version resource name for admin requests.
// Expected format: namespaces/{namespace}/models/{model}/versions/{version}
func parseAdminModelVersionName(name string) (namespaceID, modelID, version string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 6 || parts[0] != "namespaces" || parts[2] != "models" || parts[4] != "versions" {
		return "", "", "", status.Errorf(codes.InvalidArgument, "invalid model version name format: %s", name)
	}
	return parts[1], parts[3], parts[5], nil
}

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

func (h *PrivateHandler) DeployModelAdmin(ctx context.Context, req *modelpb.DeployModelAdminRequest) (*modelpb.DeployModelAdminResponse, error) {

	namespaceID, modelID, versionStr, err := parseAdminModelVersionName(req.GetName())
	if err != nil {
		return nil, err
	}

	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	modelUID, err := h.service.GetModelUIDByID(ctx, ns, modelID)
	if err != nil {
		return nil, err
	}

	version := &datamodel.ModelVersion{
		Name:     ns.Name(),
		Version:  versionStr,
		Digest:   req.GetDigest(),
		ModelUID: modelUID,
	}

	if _, err := h.service.GetModelVersionAdmin(ctx, modelUID, version.Version); err != nil {
		if err := h.service.CreateModelVersionAdmin(ctx, version); err != nil {
			return nil, err
		}
	}

	if _, err := h.service.GetRepositoryTag(ctx, &modelpb.GetRepositoryTagRequest{
		Name: fmt.Sprintf("repositories/%s/%s/tags/%s", ns.NsID, modelID, version.Version),
	}); err != nil {
		return nil, err
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, modelID, pbModel.GetHardware(), versionStr, ray.Deploy); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to deploy the model: %s", err.Error()))
	}

	return &modelpb.DeployModelAdminResponse{}, nil
}

func (h *PrivateHandler) UndeployModelAdmin(ctx context.Context, req *modelpb.UndeployModelAdminRequest) (*modelpb.UndeployModelAdminResponse, error) {

	namespaceID, modelID, versionStr, err := parseAdminModelVersionName(req.GetName())
	if err != nil {
		return nil, err
	}

	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	pbModel, err := h.service.GetModelByIDAdmin(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	if err := h.service.UpdateModelInstanceAdmin(ctx, ns, modelID, pbModel.GetHardware(), versionStr, ray.Undeploy); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to undeploy the model: %s", err.Error()))
	}

	return &modelpb.UndeployModelAdminResponse{}, nil
}

// Repository Tag Management handlers

// ListRepositoryTags lists tags in a repository
func (h *PrivateHandler) ListRepositoryTags(ctx context.Context, req *modelpb.ListRepositoryTagsRequest) (*modelpb.ListRepositoryTagsResponse, error) {
	return h.service.ListRepositoryTags(ctx, req)
}

// GetRepositoryTag gets details of a repository tag
func (h *PrivateHandler) GetRepositoryTag(ctx context.Context, req *modelpb.GetRepositoryTagRequest) (*modelpb.GetRepositoryTagResponse, error) {
	return h.service.GetRepositoryTag(ctx, req)
}

// CreateRepositoryTag creates a new repository tag
func (h *PrivateHandler) CreateRepositoryTag(ctx context.Context, req *modelpb.CreateRepositoryTagRequest) (*modelpb.CreateRepositoryTagResponse, error) {
	return h.service.CreateRepositoryTag(ctx, req)
}

// DeleteRepositoryTag deletes a repository tag
func (h *PrivateHandler) DeleteRepositoryTag(ctx context.Context, req *modelpb.DeleteRepositoryTagRequest) (*modelpb.DeleteRepositoryTagResponse, error) {
	return h.service.DeleteRepositoryTag(ctx, req)
}
