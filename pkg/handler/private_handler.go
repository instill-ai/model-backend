package handler

import (
	"context"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
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

func (h *PrivateHandler) ListModelsAdmin(ctx context.Context, req *modelPB.ListModelsAdminRequest) (*modelPB.ListModelsAdminResponse, error) {
	dbModels, nextPageToken, totalSize, err := h.service.ListModelsAdmin(ctx, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelsAdminResponse{}, err
	}

	pbModels := []*modelPB.Model{}
	for _, dbModel := range dbModels {
		modelDef, err := h.service.GetModelDefinitionByUid(ctx, dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.ListModelsAdminResponse{}, err
		}

		owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
		if err != nil {
			return &modelPB.ListModelsAdminResponse{}, err
		}

		pbModels = append(pbModels, DBModelToPBModel(&modelDef, &dbModel, owner.GetName()))
	}

	resp := modelPB.ListModelsAdminResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpModelAdmin(ctx context.Context, req *modelPB.LookUpModelAdminRequest) (*modelPB.LookUpModelAdminResponse, error) {
	sUID, err := resource.GetID(req.Permalink)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	dbModel, err := h.service.GetModelByUidAdmin(ctx, uid, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel, owner.GetName())
	return &modelPB.LookUpModelAdminResponse{Model: pbModel}, nil
}

func (h *PrivateHandler) GetModelAdmin(ctx context.Context, req *modelPB.GetModelAdminRequest) (*modelPB.GetModelAdminResponse, error) {
	id, err := resource.GetID(req.Name)
	if err != nil {
		return &modelPB.GetModelAdminResponse{}, err
	}
	dbModel, err := h.service.GetModelByIdAdmin(ctx, id, req.GetView())
	if err != nil {
		return &modelPB.GetModelAdminResponse{}, err
	}
	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		return &modelPB.GetModelAdminResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUid(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.GetModelAdminResponse{}, err
	}
	pbModel := DBModelToPBModel(&modelDef, &dbModel, owner.GetName())
	return &modelPB.GetModelAdminResponse{Model: pbModel}, err
}

func (h *PrivateHandler) CheckModel(ctx context.Context, req *modelPB.CheckModelRequest) (*modelPB.CheckModelResponse, error) {
	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		return &modelPB.CheckModelResponse{}, err
	}
	ownerPermalink := "users/" + owner.GetUid()

	modelID, err := resource.GetModelID(req.Name)
	if err != nil {
		return &modelPB.CheckModelResponse{}, err
	}

	dbModel, err := h.service.GetModelById(ctx, ownerPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.CheckModelResponse{}, err
	}

	state, err := h.service.CheckModel(ctx, dbModel.UID)
	if err != nil {
		return &modelPB.CheckModelResponse{}, err
	}

	return &modelPB.CheckModelResponse{
		State: *state,
	}, nil
}
