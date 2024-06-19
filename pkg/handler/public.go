package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func makeJSONResponse(w http.ResponseWriter, st int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(st)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(st),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

func (h *PublicHandler) ListModels(ctx context.Context, req *modelpb.ListModelsRequest) (*modelpb.ListModelsResponse, error) {

	eventName := "ListModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.ListModelsResponse{}, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		// Currently, we only have a "featured" tag, so we'll only support single tag filter for now.
		filtering.DeclareIdent("tag", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.ListModelsResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.ListModelsResponse{}, err
	}
	visibility := req.GetVisibility()

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.ListModelsResponse{}, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListModels(ctx, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.ListModelsResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	resp := modelpb.ListModelsResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

type CreateNamespaceModelRequestInterface interface {
	GetModel() *modelpb.Model
	GetParent() string
}

func (h *PublicHandler) CreateUserModel(ctx context.Context, req *modelpb.CreateUserModelRequest) (resp *modelpb.CreateUserModelResponse, err error) {
	resp = &modelpb.CreateUserModelResponse{}
	resp.Model, err = h.createNamespaceModel(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationModel(ctx context.Context, req *modelpb.CreateOrganizationModelRequest) (resp *modelpb.CreateOrganizationModelResponse, err error) {
	resp = &modelpb.CreateOrganizationModelResponse{}
	resp.Model, err = h.createNamespaceModel(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespaceModel(ctx context.Context, req CreateNamespaceModelRequestInterface) (*modelpb.Model, error) {

	ctx, span := tracer.Start(ctx, "CreateNamespaceModel",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	modelToCreate := req.GetModel()

	// Set all OUTPUT_ONLY fields to zero value on the requested payload model resource
	if err := checkfield.CheckCreateOutputOnlyFields(modelToCreate, outputOnlyFields); err != nil {
		span.SetStatus(1, ErrCheckOutputOnlyFields.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(modelToCreate.GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// validate model spec
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, modelToCreate, false); err != nil {
		span.SetStatus(1, fmt.Sprintf("Model spec is invalid %v", err.Error()))
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Model spec is invalid %v", err.Error()))
	}

	modelToCreate.OwnerName = req.GetParent()

	ns, _, err := h.service.GetRscNamespaceAndNameID(modelToCreate.GetOwnerName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if _, err := h.service.GetNamespaceModelByID(ctx, ns, modelToCreate.GetId(), modelpb.View_VIEW_FULL); err == nil {
		span.SetStatus(1, "Model already existed")
		return nil, status.Errorf(codes.AlreadyExists, "Model already existed")
	}

	if modelToCreate.GetConfiguration() == nil {
		span.SetStatus(1, "Missing Configuration")
		return nil, status.Errorf(codes.InvalidArgument, "Missing Configuration")
	}

	modelDefinitionID, err := resource.GetDefinitionID(modelToCreate.GetModelDefinition())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	modelDefinition, err := h.service.GetRepository().GetModelDefinition(modelDefinitionID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	modelSpec := utils.ModelSpec{}
	if err := json.Unmarshal(modelDefinition.ModelSpec, &modelSpec); err != nil {
		span.SetStatus(1, "Could not get model schema")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// validate model configuration
	if err := datamodel.ValidateJSONSchema(modelSpec.ModelConfigurationSchema, modelToCreate.GetConfiguration(), true); err != nil {
		span.SetStatus(1, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
	}

	switch modelDefinitionID {
	case "container":
		return createContainerizedModel(h.service, ctx, modelToCreate, ns, modelDefinition)
	default:
		span.SetStatus(1, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
	}

}

type ListNamespaceModelRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() modelpb.View
	GetParent() string
	GetFilter() string
	GetVisibility() modelpb.Model_Visibility
	GetOrderBy() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserModels(ctx context.Context, req *modelpb.ListUserModelsRequest) (resp *modelpb.ListUserModelsResponse, err error) {
	resp = &modelpb.ListUserModelsResponse{}
	resp.Models, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceModels(ctx, req)

	return resp, err
}

func (h *PublicHandler) ListOrganizationModels(ctx context.Context, req *modelpb.ListOrganizationModelsRequest) (resp *modelpb.ListOrganizationModelsResponse, err error) {
	resp = &modelpb.ListOrganizationModelsResponse{}
	resp.Models, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceModels(ctx, req)

	return resp, err
}

func (h *PublicHandler) listNamespaceModels(ctx context.Context, req ListNamespaceModelRequestInterface) (models []*modelpb.Model, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespaceModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		// Currently, we only have a "featured" tag, so we'll only support single tag filter for now.
		filtering.DeclareIdent("tag", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}
	visibility := req.GetVisibility()

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListNamespaceModels(ctx, ns, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModels, nextPageToken, totalSize, nil
}

type ListNamespaceModelVersionRequestInterface interface {
	GetPage() int32
	GetPageSize() int32
	GetName() string
}

func (h *PublicHandler) ListUserModelVersions(ctx context.Context, req *modelpb.ListUserModelVersionsRequest) (resp *modelpb.ListUserModelVersionsResponse, err error) {
	resp = &modelpb.ListUserModelVersionsResponse{}
	resp.Versions, resp.TotalSize, resp.PageSize, resp.Page, err = h.listNamespaceModelVersions(ctx, req)

	return resp, err
}

func (h *PublicHandler) ListOrganizationModelVersions(ctx context.Context, req *modelpb.ListOrganizationModelVersionsRequest) (resp *modelpb.ListOrganizationModelVersionsResponse, err error) {
	resp = &modelpb.ListOrganizationModelVersionsResponse{}
	resp.Versions, resp.TotalSize, resp.PageSize, resp.Page, err = h.listNamespaceModelVersions(ctx, req)

	return resp, err
}

func (h *PublicHandler) listNamespaceModelVersions(ctx context.Context, req ListNamespaceModelVersionRequestInterface) (versions []*modelpb.ModelVersion, totalSize int32, pageSize int32, page int32, err error) {

	eventName := "ListNamespaceModelVersions"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, 0, 0, 0, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, 0, 0, 0, err
	}

	pbModelVersions, totalSize, pageSize, page, err := h.service.ListNamespaceModelVersions(ctx, ns, req.GetPage(), req.GetPageSize(), modelID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, 0, 0, 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModelVersions, totalSize, pageSize, page, nil
}

func (h *PublicHandler) LookUpModel(ctx context.Context, req *modelpb.LookUpModelRequest) (*modelpb.LookUpModelResponse, error) {

	eventName := "LookUpModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	modelUID, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.LookUpModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.LookUpModelResponse{}, err
	}

	pbModel, err := h.service.GetModelByUID(ctx, modelUID, parseView(req.GetView()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelpb.LookUpModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.LookUpModelResponse{Model: pbModel}, nil
}

type GetNamespaceModelRequestInterface interface {
	GetName() string
	GetView() modelpb.View
}

func (h *PublicHandler) GetUserModel(ctx context.Context, req *modelpb.GetUserModelRequest) (resp *modelpb.GetUserModelResponse, err error) {
	resp = &modelpb.GetUserModelResponse{}
	resp.Model, err = h.getNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) GetOrganizationModel(ctx context.Context, req *modelpb.GetOrganizationModelRequest) (resp *modelpb.GetOrganizationModelResponse, err error) {
	resp = &modelpb.GetOrganizationModelResponse{}
	resp.Model, err = h.getNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) getNamespaceModel(ctx context.Context, req GetNamespaceModelRequestInterface) (*modelpb.Model, error) {

	eventName := "GetNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, modelID, parseView(req.GetView()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, err
}

type UpdateNamespaceModelRequestInterface interface {
	GetModel() *modelpb.Model
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserModel(ctx context.Context, req *modelpb.UpdateUserModelRequest) (resp *modelpb.UpdateUserModelResponse, err error) {
	resp = &modelpb.UpdateUserModelResponse{}
	resp.Model, err = h.updateNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) UpdateOrganizationModel(ctx context.Context, req *modelpb.UpdateOrganizationModelRequest) (resp *modelpb.UpdateOrganizationModelResponse, err error) {
	resp = &modelpb.UpdateOrganizationModelResponse{}
	resp.Model, err = h.updateNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) updateNamespaceModel(ctx context.Context, req UpdateNamespaceModelRequestInterface) (*modelpb.Model, error) {

	eventName := "UpdateNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetModel().GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel := req.GetModel()
	pbUpdateMask := req.GetUpdateMask()

	// metadata field is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Contains(path, "metadata") {
			pbUpdateMask.Paths[idx] = "metadata"
		}
	}
	if !pbUpdateMask.IsValid(pbModel) {
		return nil, status.Error(codes.InvalidArgument, "The update_mask is invalid")
	}

	var pbModelToUpdate *modelpb.Model
	if ns.NsType == resource.User {
		getResp, err := h.GetUserModel(ctx, &modelpb.GetUserModelRequest{Name: pbModel.GetName(), View: modelpb.View_VIEW_FULL.Enum()})
		if err != nil {
			span.SetStatus(1, err.Error())
			return nil, err
		}
		pbModelToUpdate = getResp.GetModel()
	}
	if ns.NsType == resource.Organization {
		getResp, err := h.GetOrganizationModel(ctx, &modelpb.GetOrganizationModelRequest{Name: pbModel.GetName(), View: modelpb.View_VIEW_FULL.Enum()})
		if err != nil {
			span.SetStatus(1, err.Error())
			return nil, err
		}
		pbModelToUpdate = getResp.GetModel()
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
	if err != nil {
		span.SetStatus(1, ErrCheckOutputOnlyFields.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, ErrFieldMask.Error())
		return nil, ErrFieldMask
	}

	if mask.IsEmpty() {
		return nil, ErrUpdateMask
	}

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbModel, pbModelToUpdate, immutableFields); err != nil {
		span.SetStatus(1, ErrCheckUpdateImmutableFields.Error())
		return nil, ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbModelToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbModel, pbModelToUpdate)
	if err != nil {
		span.SetStatus(1, ErrFieldMask.Error())
		return nil, ErrFieldMask
	}

	pbModelResp, err := h.service.UpdateNamespaceModelByID(ctx, ns, modelID, pbModelToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModelResp),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModelResp, err
}

type DeleteNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserModel(ctx context.Context, req *modelpb.DeleteUserModelRequest) (resp *modelpb.DeleteUserModelResponse, err error) {
	resp = &modelpb.DeleteUserModelResponse{}
	err = h.deleteNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) DeleteOrganizationModel(ctx context.Context, req *modelpb.DeleteOrganizationModelRequest) (resp *modelpb.DeleteOrganizationModelResponse, err error) {
	resp = &modelpb.DeleteOrganizationModelResponse{}
	err = h.deleteNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) deleteNamespaceModel(ctx context.Context, req DeleteNamespaceModelRequestInterface) error {

	eventName := "DeleteNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// Manually set the custom header to have a StatusNoContent http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	if err := h.service.DeleteNamespaceModelByID(ctx, ns, modelID); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done. resource id: %s", eventName, modelID)),
	)))

	return nil
}

type RenameNamespaceModelRequestInterface interface {
	GetName() string
	GetNewModelId() string
}

func (h *PublicHandler) RenameUserModel(ctx context.Context, req *modelpb.RenameUserModelRequest) (resp *modelpb.RenameUserModelResponse, err error) {
	resp = &modelpb.RenameUserModelResponse{}
	resp.Model, err = h.renameNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) RenameOrganizationModel(ctx context.Context, req *modelpb.RenameOrganizationModelRequest) (resp *modelpb.RenameOrganizationModelResponse, err error) {
	resp = &modelpb.RenameOrganizationModelResponse{}
	resp.Model, err = h.renameNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) renameNamespaceModel(ctx context.Context, req RenameNamespaceModelRequestInterface) (*modelpb.Model, error) {

	eventName := "RenameNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.RenameNamespaceModelByID(ctx, ns, modelID, req.GetNewModelId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type PublishNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) PublishUserModel(ctx context.Context, req *modelpb.PublishUserModelRequest) (resp *modelpb.PublishUserModelResponse, err error) {
	resp = &modelpb.PublishUserModelResponse{}
	resp.Model, err = h.publishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) PublishOrganizationModel(ctx context.Context, req *modelpb.PublishOrganizationModelRequest) (resp *modelpb.PublishOrganizationModelResponse, err error) {
	resp = &modelpb.PublishOrganizationModelResponse{}
	resp.Model, err = h.publishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) publishNamespaceModel(ctx context.Context, req PublishNamespaceModelRequestInterface) (*modelpb.Model, error) {

	eventName := "PublishNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel.Visibility = modelpb.Model_VISIBILITY_PUBLIC

	_, err = h.service.UpdateNamespaceModelByID(ctx, ns, modelID, pbModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.GetACLClient().SetPublicModelPermission(ctx, uuid.FromStringOrNil(pbModel.Uid)); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type UnpublishNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) UnpublishUserModel(ctx context.Context, req *modelpb.UnpublishUserModelRequest) (resp *modelpb.UnpublishUserModelResponse, err error) {
	resp = &modelpb.UnpublishUserModelResponse{}
	resp.Model, err = h.unpublishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) UnpublishOrganizationModel(ctx context.Context, req *modelpb.UnpublishOrganizationModelRequest) (resp *modelpb.UnpublishOrganizationModelResponse, err error) {
	resp = &modelpb.UnpublishOrganizationModelResponse{}
	resp.Model, err = h.unpublishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) unpublishNamespaceModel(ctx context.Context, req UnpublishNamespaceModelRequestInterface) (*modelpb.Model, error) {

	eventName := "UnpublishNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel.Visibility = modelpb.Model_VISIBILITY_PRIVATE

	_, err = h.service.UpdateNamespaceModelByID(ctx, ns, modelID, pbModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.GetACLClient().DeletePublicModelPermission(ctx, uuid.FromStringOrNil(pbModel.GetUid())); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type WatchNamespaceModelRequestInterface interface {
	GetName() string
	GetVersion() string
}

func (h *PublicHandler) WatchUserModel(ctx context.Context, req *modelpb.WatchUserModelRequest) (resp *modelpb.WatchUserModelResponse, err error) {
	resp = &modelpb.WatchUserModelResponse{}
	resp.State, resp.Message, err = h.watchNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) WatchOrganizationModel(ctx context.Context, req *modelpb.WatchOrganizationModelRequest) (resp *modelpb.WatchOrganizationModelResponse, err error) {
	resp = &modelpb.WatchOrganizationModelResponse{}
	resp.State, resp.Message, err = h.watchNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) WatchUserLatestModel(ctx context.Context, req *modelpb.WatchUserLatestModelRequest) (resp *modelpb.WatchUserLatestModelResponse, err error) {
	resp = &modelpb.WatchUserLatestModelResponse{}

	r := &modelpb.WatchUserModelRequest{
		Name: req.GetName(),
	}

	resp.State, resp.Message, err = h.watchNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) WatchOrganizationLatestModel(ctx context.Context, req *modelpb.WatchOrganizationLatestModelRequest) (resp *modelpb.WatchOrganizationLatestModelResponse, err error) {
	resp = &modelpb.WatchOrganizationLatestModelResponse{}

	r := &modelpb.WatchOrganizationModelRequest{
		Name: req.GetName(),
	}

	resp.State, resp.Message, err = h.watchNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) watchNamespaceModel(ctx context.Context, req WatchNamespaceModelRequestInterface) (modelpb.State, string, error) {

	eventName := "WatchNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return modelpb.State_STATE_ERROR, "", err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			ctx,
			span,
			logUUID.String(),
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelpb.State_STATE_ERROR, "", err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, modelID, modelpb.View_VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		return modelpb.State_STATE_ERROR, "", err
	}

	versionID := req.GetVersion()
	if versionID == "" {
		version, err := h.service.GetRepository().GetLatestModelVersionByModelUID(ctx, uuid.FromStringOrNil(pbModel.Uid))
		if err != nil {
			return modelpb.State_STATE_ERROR, "", err
		}
		versionID = version.Version
	}

	state, message, err := h.service.WatchModel(ctx, ns, modelID, versionID)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			ctx,
			span,
			logUUID.String(),
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelpb.State_STATE_ERROR, "", err
	}

	return *state, message, nil
}

type GetNamespaceModelCardRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) GetUserModelCard(ctx context.Context, req *modelpb.GetUserModelCardRequest) (resp *modelpb.GetUserModelCardResponse, err error) {
	resp = &modelpb.GetUserModelCardResponse{}
	resp.Readme, err = h.getNamespaceModelCard(ctx, req)

	return resp, err

}

func (h *PublicHandler) GetOrganizationModelCard(ctx context.Context, req *modelpb.GetOrganizationModelCardRequest) (resp *modelpb.GetOrganizationModelCardResponse, err error) {
	resp = &modelpb.GetOrganizationModelCardResponse{}
	resp.Readme, err = h.getNamespaceModelCard(ctx, req)

	return resp, err
}

func (h *PublicHandler) getNamespaceModelCard(ctx context.Context, req GetNamespaceModelCardRequestInterface) (*modelpb.ModelCard, error) {

	eventName := "GetNamespaceModelCard"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md", config.Config.RayServer.ModelStore, ns.Permalink(), modelID)
	stat, err := os.Stat(readmeFilePath)
	if err != nil { // return empty content base64
		span.SetStatus(1, err.Error())
		return &modelpb.ModelCard{
			Name:     req.GetName(),
			Size:     0,
			Type:     "file",
			Encoding: "base64",
			Content:  []byte(""),
		}, nil
	}

	content, err := os.ReadFile(readmeFilePath)
	if err != nil {
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.ModelCard{
		Name:     req.GetName(),
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Content:  content,
	}, nil
}

func (h *PublicHandler) GetModelDefinition(ctx context.Context, req *modelpb.GetModelDefinitionRequest) (*modelpb.GetModelDefinitionResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	definitionID, err := resource.GetDefinitionID(req.Name)
	if err != nil {
		return &modelpb.GetModelDefinitionResponse{}, err
	}

	pbModelDefinition, err := h.service.GetModelDefinition(ctx, definitionID)
	if err != nil {
		return &modelpb.GetModelDefinitionResponse{}, err
	}

	return &modelpb.GetModelDefinitionResponse{ModelDefinition: pbModelDefinition}, nil
}

func (h *PublicHandler) ListModelDefinitions(ctx context.Context, req *modelpb.ListModelDefinitionsRequest) (*modelpb.ListModelDefinitionsResponse, error) {

	ctx, span := tracer.Start(ctx, "ListModelDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	pbModelDefinitions, totalSize, nextPageToken, err := h.service.ListModelDefinitions(ctx, parseView(req.GetView()), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return &modelpb.ListModelDefinitionsResponse{}, err
	}

	resp := modelpb.ListModelDefinitionsResponse{
		ModelDefinitions: pbModelDefinitions,
		NextPageToken:    nextPageToken,
		TotalSize:        totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) ListAvailableRegions(ctx context.Context, req *modelpb.ListAvailableRegionsRequest) (*modelpb.ListAvailableRegionsResponse, error) {

	_, span := tracer.Start(ctx, "ListAvailableRegions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	regionsStruct := datamodel.RegionHardwareJSON.Properties.Region.OneOf
	hardwaresStruct := datamodel.RegionHardwareJSON.AllOf

	var regions []*modelpb.Region

	for _, r := range regionsStruct {
		subRegion := &modelpb.Region{
			RegionName: r.Const,
			Hardware:   []string{},
		}
		for _, h := range hardwaresStruct {
			if h.If.Properties.Region.Const == r.Const {
				for _, hardware := range h.Then.Properties.Hardware.OneOf {
					subRegion.Hardware = append(subRegion.Hardware, hardware.Const)
				}
				for _, hardware := range h.Then.Properties.Hardware.AnyOf {
					if hardware.Const != "" {
						subRegion.Hardware = append(subRegion.Hardware, hardware.Const)
					} else if hardware.Title != "" {
						subRegion.Hardware = append(subRegion.Hardware, hardware.Title)
					}
				}
			}
		}
		regions = append(regions, subRegion)
	}

	return &modelpb.ListAvailableRegionsResponse{
		Regions: regions,
	}, nil
}
