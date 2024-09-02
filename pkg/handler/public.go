package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/utils"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"

	"github.com/instill-ai/x/checkfield"

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
		filtering.DeclareIdent("numberOfRuns", filtering.TypeInt),
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

func (h *PublicHandler) CreateUserModel(ctx context.Context, req *modelpb.CreateUserModelRequest) (resp *modelpb.CreateUserModelResponse, err error) {
	r, err := h.CreateNamespaceModel(ctx, &modelpb.CreateNamespaceModelRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Model:       req.Model,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.CreateUserModelResponse{Model: r.Model}, nil
}

func (h *PublicHandler) CreateOrganizationModel(ctx context.Context, req *modelpb.CreateOrganizationModelRequest) (resp *modelpb.CreateOrganizationModelResponse, err error) {
	r, err := h.CreateNamespaceModel(ctx, &modelpb.CreateNamespaceModelRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Model:       req.Model,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.CreateOrganizationModelResponse{Model: r.Model}, nil
}

func (h *PublicHandler) CreateNamespaceModel(ctx context.Context, req *modelpb.CreateNamespaceModelRequest) (*modelpb.CreateNamespaceModelResponse, error) {

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

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
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
		return createContainerizedModel(h.service, ctx, ns, modelDefinition, modelToCreate)
	default:
		span.SetStatus(1, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
	}
}

func (h *PublicHandler) ListUserModels(ctx context.Context, req *modelpb.ListUserModelsRequest) (resp *modelpb.ListUserModelsResponse, err error) {
	r, err := h.ListNamespaceModels(ctx, &modelpb.ListNamespaceModelsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Visibility:  req.Visibility,
		Filter:      req.Filter,
		OrderBy:     req.OrderBy,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &modelpb.ListUserModelsResponse{
		Models:        r.Models,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

func (h *PublicHandler) ListOrganizationModels(ctx context.Context, req *modelpb.ListOrganizationModelsRequest) (resp *modelpb.ListOrganizationModelsResponse, err error) {
	r, err := h.ListNamespaceModels(ctx, &modelpb.ListNamespaceModelsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Visibility:  req.Visibility,
		Filter:      req.Filter,
		OrderBy:     req.OrderBy,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &modelpb.ListOrganizationModelsResponse{
		Models:        r.Models,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

func (h *PublicHandler) ListNamespaceModels(ctx context.Context, req *modelpb.ListNamespaceModelsRequest) (*modelpb.ListNamespaceModelsResponse, error) {

	eventName := "ListNamespaceModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		// Currently, we only have a "featured" tag, so we'll only support single tag filter for now.
		filtering.DeclareIdent("tag", filtering.TypeString),
		filtering.DeclareIdent("numberOfRuns", filtering.TypeInt),
		filtering.DeclareIdent("description", filtering.TypeString),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	visibility := req.GetVisibility()

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListNamespaceModels(ctx, ns, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.ListNamespaceModelsResponse{
		Models:        pbModels,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (h *PublicHandler) ListUserModelVersions(ctx context.Context, req *modelpb.ListUserModelVersionsRequest) (resp *modelpb.ListUserModelVersionsResponse, err error) {
	r, err := h.ListNamespaceModelVersions(ctx, &modelpb.ListNamespaceModelVersionsRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		PageSize:    req.PageSize,
		Page:        req.Page,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.ListUserModelVersionsResponse{
		Versions:  r.Versions,
		TotalSize: r.TotalSize,
		PageSize:  r.PageSize,
		Page:      r.Page,
	}, nil
}

func (h *PublicHandler) ListOrganizationModelVersions(ctx context.Context, req *modelpb.ListOrganizationModelVersionsRequest) (resp *modelpb.ListOrganizationModelVersionsResponse, err error) {
	r, err := h.ListNamespaceModelVersions(ctx, &modelpb.ListNamespaceModelVersionsRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		PageSize:    req.PageSize,
		Page:        req.Page,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.ListOrganizationModelVersionsResponse{
		Versions:  r.Versions,
		TotalSize: r.TotalSize,
		PageSize:  r.PageSize,
		Page:      r.Page,
	}, nil
}

func (h *PublicHandler) ListNamespaceModelVersions(ctx context.Context, req *modelpb.ListNamespaceModelVersionsRequest) (resp *modelpb.ListNamespaceModelVersionsResponse, err error) {

	eventName := "ListNamespaceModelVersions"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModelVersions, totalSize, pageSize, page, err := h.service.ListNamespaceModelVersions(ctx, ns, req.GetPage(), req.GetPageSize(), req.ModelId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.ListNamespaceModelVersionsResponse{
		Versions:  pbModelVersions,
		TotalSize: totalSize,
		PageSize:  pageSize,
		Page:      page,
	}, nil
}

func (h *PublicHandler) DeleteUserModelVersion(ctx context.Context, req *modelpb.DeleteUserModelVersionRequest) (*modelpb.DeleteUserModelVersionResponse, error) {
	_, err := h.DeleteNamespaceModelVersion(ctx, &modelpb.DeleteNamespaceModelVersionRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.DeleteUserModelVersionResponse{}, nil
}

func (h *PublicHandler) DeleteOrganizationModelVersion(ctx context.Context, req *modelpb.DeleteOrganizationModelVersionRequest) (*modelpb.DeleteOrganizationModelVersionResponse, error) {
	_, err := h.DeleteNamespaceModelVersion(ctx, &modelpb.DeleteNamespaceModelVersionRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.DeleteOrganizationModelVersionResponse{}, nil
}

func (h *PublicHandler) DeleteNamespaceModelVersion(ctx context.Context, req *modelpb.DeleteNamespaceModelVersionRequest) (*modelpb.DeleteNamespaceModelVersionResponse, error) {
	eventName := "DeleteNamespaceModelVersion"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteModelVersionByID(ctx, ns, req.ModelId, req.GetVersion()); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.DeleteNamespaceModelVersionResponse{}, nil
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
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.LookUpModelResponse{Model: pbModel}, nil
}

type GetNamespaceModelRequestInterface interface {
	GetName() string
	GetView() modelpb.View
}

func (h *PublicHandler) GetUserModel(ctx context.Context, req *modelpb.GetUserModelRequest) (*modelpb.GetUserModelResponse, error) {
	r, err := h.GetNamespaceModel(ctx, &modelpb.GetNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.GetUserModelResponse{
		Model: r.Model,
	}, nil
}

func (h *PublicHandler) GetOrganizationModel(ctx context.Context, req *modelpb.GetOrganizationModelRequest) (*modelpb.GetOrganizationModelResponse, error) {
	r, err := h.GetNamespaceModel(ctx, &modelpb.GetNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.GetOrganizationModelResponse{
		Model: r.Model,
	}, nil
}

func (h *PublicHandler) GetNamespaceModel(ctx context.Context, req *modelpb.GetNamespaceModelRequest) (*modelpb.GetNamespaceModelResponse, error) {

	eventName := "GetNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, req.ModelId, parseView(req.GetView()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel.Id),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.GetNamespaceModelResponse{Model: pbModel}, err
}

func (h *PublicHandler) UpdateUserModel(ctx context.Context, req *modelpb.UpdateUserModelRequest) (*modelpb.UpdateUserModelResponse, error) {
	r, err := h.UpdateNamespaceModel(ctx, &modelpb.UpdateNamespaceModelRequest{
		NamespaceId: strings.Split(req.Model.Name, "/")[1],
		ModelId:     strings.Split(req.Model.Name, "/")[3],
		Model:       req.Model,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.UpdateUserModelResponse{
		Model: r.Model,
	}, nil
}

func (h *PublicHandler) UpdateOrganizationModel(ctx context.Context, req *modelpb.UpdateOrganizationModelRequest) (*modelpb.UpdateOrganizationModelResponse, error) {
	r, err := h.UpdateNamespaceModel(ctx, &modelpb.UpdateNamespaceModelRequest{
		NamespaceId: strings.Split(req.Model.Name, "/")[1],
		ModelId:     strings.Split(req.Model.Name, "/")[3],
		Model:       req.Model,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.UpdateOrganizationModelResponse{
		Model: r.Model,
	}, nil
}

func (h *PublicHandler) UpdateNamespaceModel(ctx context.Context, req *modelpb.UpdateNamespaceModelRequest) (*modelpb.UpdateNamespaceModelResponse, error) {

	eventName := "UpdateNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
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

	getResp, err := h.GetNamespaceModel(ctx, &modelpb.GetNamespaceModelRequest{NamespaceId: req.NamespaceId, ModelId: req.ModelId, View: modelpb.View_VIEW_FULL.Enum()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	pbModelToUpdate := getResp.GetModel()

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

	pbUpdatedModel, err := h.service.UpdateNamespaceModelByID(ctx, ns, req.ModelId, pbModelToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.UpdateNamespaceModelResponse{Model: pbUpdatedModel}, err
}

type DeleteNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserModel(ctx context.Context, req *modelpb.DeleteUserModelRequest) (*modelpb.DeleteUserModelResponse, error) {
	_, err := h.DeleteNamespaceModel(ctx, &modelpb.DeleteNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.DeleteUserModelResponse{}, nil
}

func (h *PublicHandler) DeleteOrganizationModel(ctx context.Context, req *modelpb.DeleteOrganizationModelRequest) (*modelpb.DeleteOrganizationModelResponse, error) {
	_, err := h.DeleteNamespaceModel(ctx, &modelpb.DeleteNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.DeleteOrganizationModelResponse{}, nil
}

func (h *PublicHandler) DeleteNamespaceModel(ctx context.Context, req *modelpb.DeleteNamespaceModelRequest) (*modelpb.DeleteNamespaceModelResponse, error) {

	eventName := "DeleteNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Manually set the custom header to have a StatusNoContent http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteNamespaceModelByID(ctx, ns, req.ModelId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done. resource id: %s", eventName, req.ModelId)),
	)))

	return &modelpb.DeleteNamespaceModelResponse{}, nil
}

type RenameNamespaceModelRequestInterface interface {
	GetName() string
	GetNewModelId() string
}

func (h *PublicHandler) RenameUserModel(ctx context.Context, req *modelpb.RenameUserModelRequest) (*modelpb.RenameUserModelResponse, error) {
	r, err := h.RenameNamespaceModel(ctx, &modelpb.RenameNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		NewModelId:  req.NewModelId,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.RenameUserModelResponse{Model: r.Model}, nil
}

func (h *PublicHandler) RenameOrganizationModel(ctx context.Context, req *modelpb.RenameOrganizationModelRequest) (*modelpb.RenameOrganizationModelResponse, error) {
	r, err := h.RenameNamespaceModel(ctx, &modelpb.RenameNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		NewModelId:  req.NewModelId,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.RenameOrganizationModelResponse{Model: r.Model}, nil
}

func (h *PublicHandler) RenameNamespaceModel(ctx context.Context, req *modelpb.RenameNamespaceModelRequest) (*modelpb.RenameNamespaceModelResponse, error) {

	eventName := "RenameNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.RenameNamespaceModelByID(ctx, ns, req.ModelId, req.GetNewModelId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbModel.Id),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelpb.RenameNamespaceModelResponse{Model: pbModel}, nil
}

func (h *PublicHandler) WatchUserModel(ctx context.Context, req *modelpb.WatchUserModelRequest) (*modelpb.WatchUserModelResponse, error) {
	r, err := h.WatchNamespaceModel(ctx, &modelpb.WatchNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.WatchUserModelResponse{
		State:   r.State,
		Message: r.Message,
	}, nil
}

func (h *PublicHandler) WatchOrganizationModel(ctx context.Context, req *modelpb.WatchOrganizationModelRequest) (*modelpb.WatchOrganizationModelResponse, error) {
	r, err := h.WatchNamespaceModel(ctx, &modelpb.WatchNamespaceModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.WatchOrganizationModelResponse{
		State:   r.State,
		Message: r.Message,
	}, nil
}

func (h *PublicHandler) WatchUserLatestModel(ctx context.Context, req *modelpb.WatchUserLatestModelRequest) (*modelpb.WatchUserLatestModelResponse, error) {
	r, err := h.WatchNamespaceLatestModel(ctx, &modelpb.WatchNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.WatchUserLatestModelResponse{
		State:   r.State,
		Message: r.Message,
	}, nil
}

func (h *PublicHandler) WatchOrganizationLatestModel(ctx context.Context, req *modelpb.WatchOrganizationLatestModelRequest) (*modelpb.WatchOrganizationLatestModelResponse, error) {
	r, err := h.WatchNamespaceLatestModel(ctx, &modelpb.WatchNamespaceLatestModelRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		ModelId:     strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}

	return &modelpb.WatchOrganizationLatestModelResponse{
		State:   r.State,
		Message: r.Message,
	}, nil
}

type WatchNamespaceModelRequestInterface interface {
	GetNamespaceId() string
	GetModelId() string
	GetVersion() string
}

func (h *PublicHandler) WatchNamespaceModel(ctx context.Context, req *modelpb.WatchNamespaceModelRequest) (resp *modelpb.WatchNamespaceModelResponse, err error) {
	resp = &modelpb.WatchNamespaceModelResponse{}

	r := &modelpb.WatchNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
		Version:     req.GetVersion(),
	}

	resp.State, resp.Message, err = h.watchNamespaceModel(ctx, r)

	return resp, err
}

func (h *PublicHandler) WatchNamespaceLatestModel(ctx context.Context, req *modelpb.WatchNamespaceLatestModelRequest) (resp *modelpb.WatchNamespaceLatestModelResponse, err error) {
	resp = &modelpb.WatchNamespaceLatestModelResponse{}

	r := &modelpb.WatchNamespaceModelRequest{
		NamespaceId: req.GetNamespaceId(),
		ModelId:     req.GetModelId(),
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

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return modelpb.State_STATE_ERROR, "", err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			ctx,
			span,
			logUUID.String(),
			eventName,
			custom_otel.SetEventResource(req.GetModelId()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelpb.State_STATE_ERROR, "", err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, req.GetModelId(), modelpb.View_VIEW_BASIC)
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

	state, message, err := h.service.WatchModel(ctx, ns, req.GetModelId(), versionID)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			ctx,
			span,
			logUUID.String(),
			eventName,
			custom_otel.SetEventResource(req.GetModelId()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelpb.State_STATE_ERROR, "", err
	}

	return *state, message, nil
}

func (h *PublicHandler) GetModelDefinition(ctx context.Context, req *modelpb.GetModelDefinitionRequest) (*modelpb.GetModelDefinitionResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	pbModelDefinition, err := h.service.GetModelDefinition(ctx, req.ModelDefinitionId)
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

func (h *PublicHandler) ListModelRuns(ctx context.Context, req *modelpb.ListModelRunsRequest) (*modelpb.ListModelRunsResponse, error) {

	eventName := "ListModelRuns"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := custom_logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("modelVersion", filtering.TypeString),
		filtering.DeclareIdent("status", filtering.TypeString),
		filtering.DeclareIdent("source", filtering.TypeString),
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

	resp, err := h.service.ListModelTriggers(ctx, req, filter)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return resp, nil
}
