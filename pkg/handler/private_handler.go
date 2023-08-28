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
	"github.com/instill-ai/model-backend/pkg/constant"
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
		pbModel, err := h.service.DBModelToPBModel(ctx, &modelDef, dbModel)
		if err != nil {
			return &modelPB.ListModelsAdminResponse{}, err
		}
		pbModels = append(pbModels, pbModel)
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

	dbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, req.GetView())
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	modelDef, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	pbModel, err := h.service.DBModelToPBModel(ctx, &modelDef, dbModel)
	if err != nil {
		return &modelPB.LookUpModelAdminResponse{}, err
	}

	return &modelPB.LookUpModelAdminResponse{Model: pbModel}, nil
}

func (h *PrivateHandler) CheckModelAdmin(ctx context.Context, req *modelPB.CheckModelAdminRequest) (*modelPB.CheckModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.ModelPermalink)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, err
	}

	state, err := h.service.CheckModel(ctx, modelUID)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, err
	}

	return &modelPB.CheckModelAdminResponse{
		State: *state,
	}, nil
}

func (h *PrivateHandler) DeployModelAdmin(ctx context.Context, req *modelPB.DeployModelAdminRequest) (*modelPB.DeployModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.ModelPermalink)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	dbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	if !utils.HasModelInModelRepository(config.Config.TritonServer.ModelStore, dbModel.Owner, dbModel.ID) {

		modelDefinition, err := h.service.GetModelDefinitionByUID(ctx, dbModel.ModelDefinitionUid)
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		pbModel, err := h.service.DBModelToPBModel(ctx, &modelDefinition, dbModel)
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		createReq := &modelPB.CreateUserModelRequest{
			Model: pbModel,
			Parent: "users/" + constant.DefaultUserID,
		}

		var resp *modelPB.CreateUserModelResponse

		switch modelDefinition.ID {
		case "github":
			resp, err = createGitHubModel(h.service, ctx, createReq, uuid.FromStringOrNil(dbModel.Owner), &modelDefinition)
		case "artivc":
			resp, err = createArtiVCModel(h.service, ctx, createReq, uuid.FromStringOrNil(dbModel.Owner), &modelDefinition)
		case "huggingface":
			resp, err = createHuggingFaceModel(h.service, ctx, createReq, uuid.FromStringOrNil(dbModel.Owner), &modelDefinition)
		default:
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinition.ID))
		}
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "model creation error")
		}

		wfID := strings.Split(resp.Operation.Name, "/")[1]

		var operation *longrunningpb.Operation
		done := false
		for !done {
			time.Sleep(time.Second)
			operation, err = h.service.GetOperation(ctx, wfID)
			if err != nil {
				return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "get model create operation error")
			}
			done = operation.Done
		}

		if operation.GetError() != nil {
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "model create operation error")
		}

	}

	_, err = h.service.GetTritonModels(ctx, dbModel.UID)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	wfID, err := h.service.DeployUserModelAsyncAdmin(ctx, modelUID)
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

	return &modelPB.DeployModelAdminResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PrivateHandler) UndeployModelAdmin(ctx context.Context, req *modelPB.UndeployModelAdminRequest) (*modelPB.UndeployModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.ModelPermalink)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	dbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	wfId, err := h.service.UndeployUserModelAsyncAdmin(ctx, uuid.FromStringOrNil(dbModel.Owner), modelUID)
	if err != nil {
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
