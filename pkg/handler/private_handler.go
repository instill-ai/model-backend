package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type PrivateHandler struct {
	modelPB.UnimplementedModelPrivateServiceServer
	service service.Service
	triton  triton.Triton
}

func NewPrivateHandler(ctx context.Context, s service.Service, t triton.Triton) modelPB.ModelPrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
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
		modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.ListModelsAdminResponse{}, err
		}

		owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
		if err != nil {
			return &modelPB.ListModelsAdminResponse{}, err
		}

		pbModels = append(pbModels, DBModelToPBModel(ctx, &modelDef, &dbModel, GenOwnerPermalink(owner)))
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
	dbModel, err := h.service.GetModelByUIDAdmin(ctx, uid, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}
	pbModel := DBModelToPBModel(ctx, &modelDef, &dbModel, GenOwnerPermalink(owner))
	return &modelPB.LookUpModelAdminResponse{Model: pbModel}, nil
}

func (h *PrivateHandler) CheckModelAdmin(ctx context.Context, req *modelPB.CheckModelAdminRequest) (*modelPB.CheckModelAdminResponse, error) {
	sUID, err := resource.GetID(req.ModelPermalink)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	state, err := h.service.CheckModel(ctx, uid)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, err
	}

	return &modelPB.CheckModelAdminResponse{
		State: *state,
	}, nil
}

func (h *PrivateHandler) DeployModelAdmin(ctx context.Context, req *modelPB.DeployModelAdminRequest) (*modelPB.DeployModelAdminResponse, error) {

	sUID, err := resource.GetID(req.ModelPermalink)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	dbModel, err := h.service.GetModelByUIDAdmin(ctx, uid, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	if !utils.HasModelInModelRepository(config.Config.TritonServer.ModelStore, dbModel.Owner, dbModel.ID) {

		modelDefinition, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		pbModel := DBModelToPBModel(ctx, &modelDefinition, &dbModel, dbModel.Owner)

		createReq := &modelPB.CreateModelRequest{
			Model: pbModel,
		}

		var resp *modelPB.CreateModelResponse

		switch modelDefinition.ID {
		case "github":
			resp, err = createGitHubModel(h.service, ctx, createReq, dbModel.Owner, &modelDefinition)
		case "artivc":
			resp, err = createArtiVCModel(h.service, ctx, createReq, dbModel.Owner, &modelDefinition)
		case "huggingface":
			resp, err = createHuggingFaceModel(h.service, ctx, createReq, dbModel.Owner, &modelDefinition)
		default:
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinition.ID))
		}
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "model creation error")
		}

		wfID := strings.Split(resp.Operation.Name, "/")[1]

		if err := h.service.UpdateResourceState(
			ctx,
			dbModel.UID,
			modelPB.Model_STATE_UNSPECIFIED,
			nil,
			&wfID,
		); err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		done := false
		for !done {
			operation, _ := h.service.GetOperation(ctx, wfID)
			done = operation.Done
			time.Sleep(time.Second)
		}

	}

	_, err = h.service.GetTritonModels(ctx, dbModel.UID)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	wfID, err := h.service.DeployModelAsync(ctx, dbModel.Owner, dbModel.UID)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] deploy a model error: %s", err.Error()),
			"triton-inference-server",
			"deploy model",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] deploy model error",
				"triton-inference-server",
				"Out of memory for deploying the model to triton server, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			return &modelPB.DeployModelAdminResponse{}, fmt.Errorf(e.Error())
		}
		return &modelPB.DeployModelAdminResponse{}, st.Err()
	}

	if err := h.service.UpdateResourceState(
		ctx,
		dbModel.UID,
		modelPB.Model_STATE_UNSPECIFIED,
		nil,
		&wfID,
	); err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	return &modelPB.DeployModelAdminResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PrivateHandler) UndeployModelAdmin(ctx context.Context, req *modelPB.UndeployModelAdminRequest) (*modelPB.UndeployModelAdminResponse, error) {

	sUID, err := resource.GetID(req.ModelPermalink)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}
	uid, err := uuid.FromString(sUID)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	dbModel, err := h.service.GetModelByUIDAdmin(ctx, uid, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	wfId, err := h.service.UndeployModelAsync(ctx, dbModel.Owner, dbModel.UID)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	if err := h.service.UpdateResourceState(
		ctx,
		dbModel.UID,
		modelPB.Model_STATE_UNSPECIFIED,
		nil,
		&wfId,
	); err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	return &modelPB.UndeployModelAdminResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfId),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}
