package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
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

func (h *PrivateHandler) CheckModelAdmin(ctx context.Context, req *modelPB.CheckModelAdminRequest) (*modelPB.CheckModelAdminResponse, error) {

	modelUID, err := resource.GetRscPermalinkUID(req.ModelPermalink)
	if err != nil {
		return &modelPB.CheckModelAdminResponse{}, err
	}

	state, err := h.service.CheckModelAdmin(ctx, modelUID)
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

	pbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.DeployModelAdminResponse{}, err
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(pbModel.GetOwnerName())
	if err != nil {
		return nil, err
	}

	var authUser *service.AuthUser
	if ns.NsType == resource.Organization {
		resp, err := h.service.GetMgmtPrivateServiceClient().GetOrganizationAdmin(ctx, &mgmtPB.GetOrganizationAdminRequest{
			Name: ns.Name(),
		})
		if err != nil {
			return nil, err
		}
		orgOwnerNS, _, err := h.service.GetRscNamespaceAndNameID(resp.GetOrganization().GetOwner().GetName())
		if err != nil {
			return nil, err
		}

		authUser = &service.AuthUser{
			UID:       orgOwnerNS.NsUID,
			IsVisitor: false,
		}
	} else {
		authUser = &service.AuthUser{
			UID:       ns.NsUID,
			IsVisitor: false,
		}
	}

	if !utils.HasModelInModelRepository(config.Config.RayServer.ModelStore, ns.Permalink(), pbModel.Id) {

		modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		modelDefinition, err := h.service.GetRepository().GetModelDefinition(modelDefID)
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, err
		}

		var operation *longrunningpb.Operation

		switch modelDefinition.ID {
		case "github":
			operation, err = createGitHubModel(h.service, ctx, pbModel, ns, authUser, modelDefinition)
		case "artivc":
			operation, err = createArtiVCModel(h.service, ctx, pbModel, ns, authUser, modelDefinition)
		case "huggingface":
			operation, err = createHuggingFaceModel(h.service, ctx, pbModel, ns, authUser, modelDefinition)
		default:
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinition.ID))
		}
		if err != nil {
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, fmt.Sprintf("model creation error: %v", err))
		}

		done := false
		for !done {
			time.Sleep(time.Second)
			operation, err = h.service.GetOperation(ctx, strings.Split(operation.Name, "/")[1])
			if err != nil {
				return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "get model create operation error")
			}
			done = operation.Done
		}

		if operation.GetError() != nil {
			return &modelPB.DeployModelAdminResponse{}, status.Errorf(codes.Internal, "model create operation error")
		}

	}

	wfID, err := h.service.DeployNamespaceModelAsyncAdmin(ctx, ns.NsID, modelUID)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] deploy a model error: %s", err.Error()),
			"ray-server",
			"deploy model",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] deploy model error",
				"ray-server",
				"Out of memory for deploying the model to ray server, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			return &modelPB.DeployModelAdminResponse{}, errors.New(e.Error())
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

	pbModel, err := h.service.GetModelByUIDAdmin(ctx, modelUID, modelPB.View_VIEW_FULL)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(pbModel.GetOwnerName())
	if err != nil {
		return nil, err
	}

	wfID, err := h.service.UndeployNamespaceModelAsyncAdmin(ctx, ns.NsID, modelUID)
	if err != nil {
		return &modelPB.UndeployModelAdminResponse{}, err
	}

	return &modelPB.UndeployModelAdminResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}}, nil
}

func (h *PrivateHandler) CheckModelNamespaceExist(ctx context.Context, req *modelPB.CheckModelNamespaceExistRequest) (*modelPB.CheckModelNamespaceExistResponse, error) {

	modelNamespace, err := resource.GetModelNamespace(req.ModelNamespace)
	if err != nil {
		return &modelPB.CheckModelNamespaceExistResponse{IsNamespaceExisted: false}, err
	}
	pbModel, err := h.service.GetRepository().GetModelByModelNamespace(ctx, modelNamespace, true)

	fmt.Println("[Tony DEBUG] CheckModelNamespaceExist, pbModel: ", pbModel)
	if err != nil {
		return &modelPB.CheckModelNamespaceExistResponse{IsNamespaceExisted: false}, nil
	}
	return &modelPB.CheckModelNamespaceExistResponse{IsNamespaceExisted: true}, err
}

// Private: Update Admin Model Namepsace Admin
// Update DB State, Write DB
// Set, Unset,
// Remove ACL
// SetModelDeployState(SetModelDeployStateRequest) returns (SetModelDeployStateResponse) {

func (h *PrivateHandler) SetModelDeployState(ctx context.Context, req *modelPB.SetModelDeployStateRequest) (*modelPB.SetModelDeployStateResponse, error) {

	eventName := "SetModelDeployState"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.SetModelDeployStateResponse{}, err
	}

	// TODO: Remove ACL Part
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.SetModelDeployStateResponse{}, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.SetModelDeployStateResponse{}, err
	}

	// set model version
	if _, err := h.service.UpdateNamespaceModelVersion(ctx, ns, authUser, pbModel, req.GetVersion()); err != nil {
		return &modelPB.SetModelDeployStateResponse{}, err
	}

	// set user desired state to STATE_ONLINE
	if _, err := h.service.UpdateNamespaceModelStateByID(ctx, ns, authUser, pbModel, modelPB.Model_STATE_ONLINE); err != nil {
		return &modelPB.SetModelDeployStateResponse{}, err
	}

	state := modelPB.Model_STATE_OFFLINE.Enum()
	for state.String() == modelPB.Model_STATE_OFFLINE.String() {
		if state, _, err = h.service.GetResourceState(ctx, uuid.FromStringOrNil(pbModel.Uid)); err != nil {
			return &modelPB.SetModelDeployStateResponse{}, err
		}
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.SetModelDeployStateResponse{ModelNamespace: modelID}, nil
}
