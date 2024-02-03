package handler

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func makeJSONResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

// HandleCreateModelByMultiPartFormData is a custom handler
func HandleCreateModelByMultiPartFormData(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	eventName := "HandleCreateModelByMultiPartFormData"

	ctx, span := tracer.Start(req.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	headers := map[string]string{}
	// inject header into ctx
	for key, value := range req.Header {
		if len(value) > 0 {
			headers[key] = value[0]
		}
	}
	md := metadata.New(headers)
	ctx = metadata.NewIncomingContext(ctx, md)

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		span.SetStatus(1, "")
		return
	}

	authUser, err := s.AuthenticateUser(ctx, false)
	if err != nil {
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			span.SetStatus(1, "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			span.SetStatus(1, "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			return
		}
	}

	parent := pathParams["parent"]

	ns, _, err := s.GetRscNamespaceAndNameID(parent)
	if err != nil {
		makeJSONResponse(w, 400, "Model path format error", "Model path format error")
		span.SetStatus(1, "Model path format error")
		return
	}

	modelID := req.FormValue("id")
	if modelID == "" {
		makeJSONResponse(w, 400, "Missing parameter", "Model Id need to be specified")
		span.SetStatus(1, "Model Id need to be specified")
		return
	}

	modelDefinitionName := req.FormValue("model_definition")
	if modelDefinitionName == "" {
		makeJSONResponse(w, 400, "Missing parameter", "modelDefinitionName need to be specified")
		span.SetStatus(1, "modelDefinitionName need to be specified")
		return
	}
	modelDefinitionID, err := resource.GetDefinitionID(modelDefinitionName)
	if err != nil {
		makeJSONResponse(w, 400, "Invalid parameter", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	viz := req.FormValue("visibility")
	var visibility modelPB.Model_Visibility
	if viz != "" {
		if utils.Visibility[viz] == modelPB.Model_VISIBILITY_UNSPECIFIED {
			makeJSONResponse(w, 400, "Invalid parameter", "Visibility is invalid")
			span.SetStatus(1, "Visibility is invalid")
			return
		} else {
			visibility = utils.Visibility[viz]
		}
	} else {
		visibility = modelPB.Model_VISIBILITY_PRIVATE
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		return
	}
	file, fileHeader, err := req.FormFile("content")
	if err != nil {
		makeJSONResponse(w, 500, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buf := bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, 1024)
	count := 0
	for {
		if count, err = reader.Read(part); err != nil {
			break
		}
		buf.Write(part[:count])
	}
	if err != io.EOF {
		makeJSONResponse(w, 400, "File Error", "Error reading input file")
		span.SetStatus(1, "Error reading input file")
		return
	}
	rdid, _ := uuid.NewV4()
	tmpFile := path.Join("/tmp", rdid.String())
	fp, err := os.Create(tmpFile)
	if err != nil {
		makeJSONResponse(w, 400, "File Error", "Error reading input file")
		span.SetStatus(1, "Error reading input file")
		return
	}
	err = utils.WriteToFp(fp, buf.Bytes())
	if err != nil {
		makeJSONResponse(w, 400, "File Error", "Error reading input file")
		span.SetStatus(1, "Error reading input file")
		return
	}

	// validate model configuration
	localModelDefinition, err := s.GetRepository().GetModelDefinition(modelDefinitionID)
	if err != nil {
		makeJSONResponse(w, 400, "Parameter invalid", "ModelDefinitionId not found")
		span.SetStatus(1, "ModelDefinitionId not found")
		return
	}
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(localModelDefinition.ModelSpec.String()), rs); err != nil {
		makeJSONResponse(w, 500, "Add Model Error", "Could not get model definition")
		span.SetStatus(1, "Could not get model definition")
		return
	}
	modelConfiguration := datamodel.LocalModelConfiguration{
		Content: fileHeader.Filename,
	}

	if err := datamodel.ValidateJSONSchema(rs, modelConfiguration, true); err != nil {
		makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		span.SetStatus(1, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		return
	}
	modelConfiguration.Tag = "latest" // Set after validation. Because the model definition do not contain tag.

	bModelConfig, _ := json.Marshal(modelConfiguration)
	var uploadedModel = datamodel.Model{
		ID:                 modelID,
		ModelDefinitionUID: localModelDefinition.UID,
		Owner:              authUser.Permalink(),
		Visibility:         datamodel.ModelVisibility(visibility),
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: req.FormValue("description"),
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	// Validate ModelDefinition JSON Schema
	pbModel, err := s.DBToPBModel(ctx, localModelDefinition, &uploadedModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return
	}
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, pbModel, true); err != nil {
		makeJSONResponse(w, 400, "Add Model Error", fmt.Sprintf("Model definition is invalid %v", err.Error()))
		span.SetStatus(1, fmt.Sprintf("Model definition is invalid %v", err.Error()))
		return
	}

	_, err = s.GetNamespaceModelByID(req.Context(), ns, authUser, uploadedModel.ID, modelPB.View_VIEW_FULL)
	if err == nil {
		makeJSONResponse(w, 409, "Add Model Error", fmt.Sprintf("The model %v already existed", uploadedModel.ID))
		span.SetStatus(1, fmt.Sprintf("The model %v already existed", uploadedModel.ID))
		return
	}

	readmeFilePath, ensembleFilePath, err := utils.Unzip(tmpFile, config.Config.TritonServer.ModelStore, authUser.Permalink(), &uploadedModel)
	_ = os.Remove(tmpFile) // remove uploaded temporary zip file
	if err != nil {
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 400, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
		return
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := utils.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
			makeJSONResponse(w, 400, "Add Model Error", err.Error())
			span.SetStatus(1, err.Error())
			return
		}
		if modelMeta.Task == "" {
			uploadedModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
		} else {
			if val, ok := utils.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				uploadedModel.Task = datamodel.ModelTask(val)
			} else {
				utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
				makeJSONResponse(w, 400, "Add Model Error", "README.md contains unsupported task")
				span.SetStatus(1, "README.md contains unsupported task")
				return
			}
		}
	} else {
		uploadedModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
	}

	maxBatchSize := 0
	if ensembleFilePath != "" {
		maxBatchSize, err = utils.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				"[handler] create a model error",
				"Local model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			obj, _ := json.Marshal(st.Details())
			makeJSONResponse(w, 400, st.Message(), string(obj))
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
			span.SetStatus(1, err.Error())
			return
		}
	}

	allowedMaxBatchSize := utils.GetSupportedBatchSize(uploadedModel.Task)

	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})
		if e != nil {
			logger.Error(e.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
		obj, _ := json.Marshal(st.Details())
		makeJSONResponse(w, 400, st.Message(), string(obj))
		span.SetStatus(1, string(obj))
		return
	}

	wfID, err := s.CreateNamespaceModelAsync(req.Context(), ns, authUser, &uploadedModel)
	if err != nil {
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 500, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(201)

	m := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: false, EmitUnpopulated: true}
	b, err := m.Marshal(&modelPB.CreateUserModelResponse{Operation: &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}})
	if err != nil {
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), uploadedModel.ID, modelConfiguration.Tag)
		makeJSONResponse(w, 500, "Add Model Error", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(uploadedModel),
	)))

	_, _ = w.Write(b)

}

func (h *PublicHandler) ListModels(ctx context.Context, req *modelPB.ListModelsRequest) (*modelPB.ListModelsResponse, error) {

	eventName := "ListModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	authUser, err := h.service.AuthenticateUser(ctx, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.ListModelsResponse{}, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListModels(ctx, authUser, int32(req.GetPageSize()), req.GetPageToken(), parseView(req.GetView()), modelPB.Model_VISIBILITY_PUBLIC.Enum(), req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.ListModelsResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModels),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	resp := modelPB.ListModelsResponse{
		Models:        pbModels,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

type CreateNamespaceModelRequestInterface interface {
	GetModel() *modelPB.Model
	GetParent() string
}

func (h *PublicHandler) CreateUserModel(ctx context.Context, req *modelPB.CreateUserModelRequest) (resp *modelPB.CreateUserModelResponse, err error) {
	resp = &modelPB.CreateUserModelResponse{}
	resp.Operation, err = h.createNamespaceModel(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationModel(ctx context.Context, req *modelPB.CreateOrganizationModelRequest) (resp *modelPB.CreateOrganizationModelResponse, err error) {
	resp = &modelPB.CreateOrganizationModelResponse{}
	resp.Operation, err = h.createNamespaceModel(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespaceModel(ctx context.Context, req CreateNamespaceModelRequestInterface) (*longrunningpb.Operation, error) {

	ctx, span := tracer.Start(ctx, "CreateNamespaceModel",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	resp := &longrunningpb.Operation{}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload model resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetModel(), outputOnlyFields); err != nil {
		span.SetStatus(1, ErrCheckOutputOnlyFields.Error())
		return resp, ErrCheckOutputOnlyFields
	}

	// Return error if REQUIRED fields are not provided in the requested payload model resource
	if err := checkfield.CheckRequiredFields(req.GetModel(), requiredFields); err != nil {
		span.SetStatus(1, ErrCheckRequiredFields.Error())
		return resp, ErrCheckRequiredFields
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.GetModel().GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// Validate ModelDefinition JSON Schema
	if err := datamodel.ValidateJSONSchema(datamodel.ModelJSONSchema, req.GetModel(), false); err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	if model, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, req.GetModel().GetId(), modelPB.View_VIEW_FULL); err == nil {
		if utils.HasModelInModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), model.Id) {
			span.SetStatus(1, "Model already existed")
			return resp, status.Errorf(codes.AlreadyExists, "Model already existed")
		}
	}

	if req.GetModel().GetConfiguration() == nil {
		span.SetStatus(1, "Missing Configuration")
		return resp, status.Errorf(codes.InvalidArgument, "Missing Configuration")
	}

	modelDefinitionID, err := resource.GetDefinitionID(req.GetModel().ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	modelDefinition, err := h.service.GetRepository().GetModelDefinition(modelDefinitionID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// validate model configuration
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(modelDefinition.ModelSpec.String()), rs); err != nil {
		span.SetStatus(1, "Could not get model definition")
		return resp, status.Errorf(codes.InvalidArgument, "Could not get model definition")
	}
	if err := datamodel.ValidateJSONSchema(rs, req.GetModel().GetConfiguration(), true); err != nil {
		span.SetStatus(1, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
		return resp, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Model configuration is invalid %v", err.Error()))
	}

	switch modelDefinitionID {
	case "github":
		return createGitHubModel(h.service, ctx, req, ns, authUser, modelDefinition)
	case "artivc":
		return createArtiVCModel(h.service, ctx, req, ns, authUser, modelDefinition)
	case "huggingface":
		return createHuggingFaceModel(h.service, ctx, req, ns, authUser, modelDefinition)
	default:
		span.SetStatus(1, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
		return resp, status.Errorf(codes.InvalidArgument, fmt.Sprintf("model definition %v is not supported", modelDefinitionID))
	}

}

type ListNamespaceModelRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() modelPB.View
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserModels(ctx context.Context, req *modelPB.ListUserModelsRequest) (resp *modelPB.ListUserModelsResponse, err error) {
	resp = &modelPB.ListUserModelsResponse{}
	resp.Models, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceModels(ctx, req)

	return resp, err
}

func (h *PublicHandler) ListOrganizationModels(ctx context.Context, req *modelPB.ListOrganizationModelsRequest) (resp *modelPB.ListOrganizationModelsResponse, err error) {
	resp = &modelPB.ListOrganizationModelsResponse{}
	resp.Models, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceModels(ctx, req)

	return resp, err
}

func (h *PublicHandler) listNamespaceModels(ctx context.Context, req ListNamespaceModelRequestInterface) (models []*modelPB.Model, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespaceModels"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	pbModels, totalSize, nextPageToken, err := h.service.ListNamespaceModels(ctx, ns, authUser, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModels),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModels, nextPageToken, totalSize, nil
}

func (h *PublicHandler) LookUpModel(ctx context.Context, req *modelPB.LookUpModelRequest) (*modelPB.LookUpModelResponse, error) {

	eventName := "LookUpModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	modelUID, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}

	pbModel, err := h.service.GetModelByUID(ctx, authUser, modelUID, parseView(req.GetView()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &modelPB.LookUpModelResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.LookUpModelResponse{Model: pbModel}, nil
}

type GetNamespaceModelRequestInterface interface {
	GetName() string
	GetView() modelPB.View
}

func (h *PublicHandler) GetUserModel(ctx context.Context, req *modelPB.GetUserModelRequest) (resp *modelPB.GetUserModelResponse, err error) {
	resp = &modelPB.GetUserModelResponse{}
	resp.Model, err = h.getNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) GetOrganizationModel(ctx context.Context, req *modelPB.GetOrganizationModelRequest) (resp *modelPB.GetOrganizationModelResponse, err error) {
	resp = &modelPB.GetOrganizationModelResponse{}
	resp.Model, err = h.getNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) getNamespaceModel(ctx context.Context, req GetNamespaceModelRequestInterface) (*modelPB.Model, error) {

	eventName := "GetNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, parseView(req.GetView()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, err
}

type UpdateNamespaceModelRequestInterface interface {
	GetModel() *modelPB.Model
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserModel(ctx context.Context, req *modelPB.UpdateUserModelRequest) (resp *modelPB.UpdateUserModelResponse, err error) {
	resp = &modelPB.UpdateUserModelResponse{}
	resp.Model, err = h.updateNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) UpdateOrganizationModel(ctx context.Context, req *modelPB.UpdateOrganizationModelRequest) (resp *modelPB.UpdateOrganizationModelResponse, err error) {
	resp = &modelPB.UpdateOrganizationModelResponse{}
	resp.Model, err = h.updateNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) updateNamespaceModel(ctx context.Context, req UpdateNamespaceModelRequestInterface) (*modelPB.Model, error) {

	eventName := "UpdateNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetModel().GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
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

	var pbModelToUpdate *modelPB.Model
	if ns.NsType == resource.User {
		getResp, err := h.GetUserModel(ctx, &modelPB.GetUserModelRequest{Name: pbModel.GetName(), View: modelPB.View_VIEW_FULL.Enum()})
		if err != nil {
			span.SetStatus(1, err.Error())
			return nil, err
		}
		pbModelToUpdate = getResp.GetModel()
	}
	if ns.NsType == resource.Organization {
		getResp, err := h.GetOrganizationModel(ctx, &modelPB.GetOrganizationModelRequest{Name: pbModel.GetName(), View: modelPB.View_VIEW_FULL.Enum()})
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

	pbModelResp, err := h.service.UpdateNamespaceModelByID(ctx, ns, authUser, modelID, pbModelToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModelResp),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModelResp, err
}

type DeleteNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserModel(ctx context.Context, req *modelPB.DeleteUserModelRequest) (resp *modelPB.DeleteUserModelResponse, err error) {
	resp = &modelPB.DeleteUserModelResponse{}
	err = h.deleteNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) DeleteOrganizationModel(ctx context.Context, req *modelPB.DeleteOrganizationModelRequest) (resp *modelPB.DeleteOrganizationModelResponse, err error) {
	resp = &modelPB.DeleteOrganizationModelResponse{}
	err = h.deleteNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) deleteNamespaceModel(ctx context.Context, req DeleteNamespaceModelRequestInterface) error {

	eventName := "DeleteNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// Manually set the custom header to have a StatusNoContent http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	if err := h.service.DeleteNamespaceModelByID(ctx, ns, authUser, modelID); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventMessage(fmt.Sprintf("%s done. resource id: %s", eventName, modelID)),
	)))

	return nil
}

type RenameNamespaceModelRequestInterface interface {
	GetName() string
	GetNewModelId() string
}

func (h *PublicHandler) RenameUserModel(ctx context.Context, req *modelPB.RenameUserModelRequest) (resp *modelPB.RenameUserModelResponse, err error) {
	resp = &modelPB.RenameUserModelResponse{}
	resp.Model, err = h.renameNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) RenameOrganizationModel(ctx context.Context, req *modelPB.RenameOrganizationModelRequest) (resp *modelPB.RenameOrganizationModelResponse, err error) {
	resp = &modelPB.RenameOrganizationModelResponse{}
	resp.Model, err = h.renameNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) renameNamespaceModel(ctx context.Context, req RenameNamespaceModelRequestInterface) (*modelPB.Model, error) {

	eventName := "RenameNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.RenameNamespaceModelByID(ctx, ns, authUser, modelID, req.GetNewModelId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type PublishNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) PublishUserModel(ctx context.Context, req *modelPB.PublishUserModelRequest) (resp *modelPB.PublishUserModelResponse, err error) {
	resp = &modelPB.PublishUserModelResponse{}
	resp.Model, err = h.publishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) PublishOrganizationModel(ctx context.Context, req *modelPB.PublishOrganizationModelRequest) (resp *modelPB.PublishOrganizationModelResponse, err error) {
	resp = &modelPB.PublishOrganizationModelResponse{}
	resp.Model, err = h.publishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) publishNamespaceModel(ctx context.Context, req PublishNamespaceModelRequestInterface) (*modelPB.Model, error) {

	eventName := "PublishNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel.Visibility = modelPB.Model_VISIBILITY_PUBLIC

	_, err = h.service.UpdateNamespaceModelByID(ctx, ns, authUser, modelID, pbModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	err = h.service.GetACLClient().SetPublicModelPermission(uuid.FromStringOrNil(pbModel.GetUid()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type UnpublishNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) UnpublishUserModel(ctx context.Context, req *modelPB.UnpublishUserModelRequest) (resp *modelPB.UnpublishUserModelResponse, err error) {
	resp = &modelPB.UnpublishUserModelResponse{}
	resp.Model, err = h.unpublishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) UnpublishOrganizationModel(ctx context.Context, req *modelPB.UnpublishOrganizationModelRequest) (resp *modelPB.UnpublishOrganizationModelResponse, err error) {
	resp = &modelPB.UnpublishOrganizationModelResponse{}
	resp.Model, err = h.unpublishNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) unpublishNamespaceModel(ctx context.Context, req UnpublishNamespaceModelRequestInterface) (*modelPB.Model, error) {

	eventName := "UnpublishNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbModel.Visibility = modelPB.Model_VISIBILITY_PRIVATE

	_, err = h.service.UpdateNamespaceModelByID(ctx, ns, authUser, modelID, pbModel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	err = h.service.GetACLClient().DeletePublicModelPermission(uuid.FromStringOrNil(pbModel.GetUid()))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return pbModel, nil
}

type DeployNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeployUserModel(ctx context.Context, req *modelPB.DeployUserModelRequest) (resp *modelPB.DeployUserModelResponse, err error) {
	resp = &modelPB.DeployUserModelResponse{}
	resp.ModelId, err = h.deployNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) DeployOrganizationModel(ctx context.Context, req *modelPB.DeployOrganizationModelRequest) (resp *modelPB.DeployOrganizationModelResponse, err error) {
	resp = &modelPB.DeployOrganizationModelResponse{}
	resp.ModelId, err = h.deployNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) deployNamespaceModel(ctx context.Context, req DeployNamespaceModelRequestInterface) (string, error) {

	eventName := "DeployNamespaceModel"

	// block for controller to update the state
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	_, err = h.service.GetInferenceModels(ctx, uuid.FromStringOrNil(pbModel.Uid))
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	// set user desired state to STATE_ONLINE
	if _, err := h.service.UpdateNamespaceModelStateByID(ctx, ns, authUser, pbModel, modelPB.Model_STATE_ONLINE); err != nil {
		return "", err
	}

	state := modelPB.Model_STATE_OFFLINE.Enum()
	for state.String() == modelPB.Model_STATE_OFFLINE.String() {
		if state, _, err = h.service.GetResourceState(ctx, uuid.FromStringOrNil(pbModel.Uid)); err != nil {
			return "", err
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

	return modelID, nil
}

type UndeployNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) UndeployUserModel(ctx context.Context, req *modelPB.UndeployUserModelRequest) (resp *modelPB.UndeployUserModelResponse, err error) {
	resp = &modelPB.UndeployUserModelResponse{}
	resp.ModelId, err = h.undeployNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) UndeployOrganizationModel(ctx context.Context, req *modelPB.UndeployOrganizationModelRequest) (resp *modelPB.UndeployOrganizationModelResponse, err error) {
	resp = &modelPB.UndeployOrganizationModelResponse{}
	resp.ModelId, err = h.undeployNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) undeployNamespaceModel(ctx context.Context, req UndeployNamespaceModelRequestInterface) (string, error) {

	eventName := "UndeployNamespaceModel"

	// block for controller to update the state
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	if err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	// set user desired state to STATE_OFFLINE
	if _, err := h.service.UpdateNamespaceModelStateByID(ctx, ns, authUser, pbModel, modelPB.Model_STATE_OFFLINE); err != nil {
		span.SetStatus(1, err.Error())
		return "", err
	}

	state := modelPB.Model_STATE_ONLINE.Enum()
	for state.String() == modelPB.Model_STATE_ONLINE.String() {
		if state, _, err = h.service.GetResourceState(ctx, uuid.FromStringOrNil(pbModel.Uid)); err != nil {
			return "", err
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

	return modelID, nil
}

type WatchNamespaceModelRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) WatchUserModel(ctx context.Context, req *modelPB.WatchUserModelRequest) (resp *modelPB.WatchUserModelResponse, err error) {
	resp = &modelPB.WatchUserModelResponse{}
	resp.State, resp.Progress, err = h.watchNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) WatchOrganizationModel(ctx context.Context, req *modelPB.WatchOrganizationModelRequest) (resp *modelPB.WatchOrganizationModelResponse, err error) {
	resp = &modelPB.WatchOrganizationModelResponse{}
	resp.State, resp.Progress, err = h.watchNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) watchNamespaceModel(ctx context.Context, req WatchNamespaceModelRequestInterface) (modelPB.Model_State, int32, error) {

	eventName := "WatchNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return modelPB.Model_STATE_ERROR, 0, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return modelPB.Model_STATE_ERROR, 0, err
	}

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelPB.Model_STATE_ERROR, 0, err
	}

	// check permission
	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelPB.Model_STATE_ERROR, 0, err
	}

	state, _, err := h.service.GetResourceState(ctx, uuid.FromStringOrNil(pbModel.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
			custom_otel.SetEventResource(req.GetName()),
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return modelPB.Model_STATE_ERROR, 0, err
	}

	return *state, 0, nil
}

type TriggerNamespaceModelRequestInterface interface {
	GetName() string
	GetTaskInputs() []*modelPB.TaskInput
}

func (h *PublicHandler) TriggerUserModel(ctx context.Context, req *modelPB.TriggerUserModelRequest) (resp *modelPB.TriggerUserModelResponse, err error) {
	resp = &modelPB.TriggerUserModelResponse{}
	resp.Task, resp.TaskOutputs, err = h.triggerNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) TriggerOrganizationModel(ctx context.Context, req *modelPB.TriggerOrganizationModelRequest) (resp *modelPB.TriggerOrganizationModelResponse, err error) {
	resp = &modelPB.TriggerOrganizationModelResponse{}
	resp.Task, resp.TaskOutputs, err = h.triggerNamespaceModel(ctx, req)

	return resp, err
}

func (h *PublicHandler) triggerNamespaceModel(ctx context.Context, req TriggerNamespaceModelRequestInterface) (commonPB.Task, []*modelPB.TaskOutput, error) {

	startTime := time.Now()
	eventName := "TriggerNamespaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonPB.Task_TASK_UNSPECIFIED, nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonPB.Task_TASK_UNSPECIFIED, nil, err
	}

	pbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonPB.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonPB.Task_TASK_UNSPECIFIED, nil, err
	}

	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	usageData := utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtPB.OwnerType_OWNER_TYPE_USER,
		UserUID:            authUser.UID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          commonPB.Task(pbModel.Task),
	}

	var inputInfer interface{}
	var lenInputs = 1
	switch commonPB.Task(pbModel.Task) {
	case commonPB.Task_TASK_CLASSIFICATION,
		commonPB.Task_TASK_DETECTION,
		commonPB.Task_TASK_INSTANCE_SEGMENTATION,
		commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
		commonPB.Task_TASK_OCR,
		commonPB.Task_TASK_KEYPOINT,
		commonPB.Task_TASK_UNSPECIFIED:
		imageInput, err := parseImageRequestInputsToBytes(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseTexToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textToImage
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		imageToImage, err := parseImageToImageRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = imageToImage
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnswering, err := parseVisualQuestionAnsweringRequestInputs(
			ctx,
			&modelPB.TriggerUserModelRequest{
				Name:       req.GetName(),
				TaskInputs: req.GetTaskInputs(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		inputInfer = visualQuestionAnswering
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChat, err := parseTexGenerationChatRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGenerationChat
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGeneration, err := parseTexGenerationRequestInputs(ctx, req)
		if err != nil {
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = h.service.WriteNewDataPoint(ctx, usageData)
			return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lenInputs = 1
		inputInfer = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		tritonModelInDB, err := h.service.GetInferenceEnsembleModel(ctx, uuid.FromStringOrNil(pbModel.Uid))
		if tritonModelInDB.Platform == "ensemble" {
			if err != nil {
				span.SetStatus(1, err.Error())
				usageData.Status = mgmtPB.Status_STATUS_ERRORED
				_ = h.service.WriteNewDataPoint(ctx, usageData)
				return commonPB.Task_TASK_UNSPECIFIED, nil, err
			}
			configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
			doSupportBatch, err := utils.DoSupportBatch(configPbFilePath)
			if err != nil {
				span.SetStatus(1, err.Error())
				usageData.Status = mgmtPB.Status_STATUS_ERRORED
				_ = h.service.WriteNewDataPoint(ctx, usageData)
				return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, err.Error())
			}
			if !doSupportBatch {
				span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
				return commonPB.Task_TASK_UNSPECIFIED, nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
			}
		}
	}
	task := commonPB.Task(pbModel.Task)
	response, err := h.service.TriggerNamespaceModelByID(ctx, ns, authUser, modelID, inputInfer, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		usageData.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewDataPoint(ctx, usageData)
		return commonPB.Task_TASK_UNSPECIFIED, nil, st.Err()
	}

	usageData.Status = mgmtPB.Status_STATUS_COMPLETED
	if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
		logger.Warn("usage and metric data write fail")
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return task, response, nil
}

func inferModelByUpload(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	startTime := time.Now()
	eventName := "InferModelByUpload"

	ctx, span := tracer.Start(req.Context(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// inject header into ctx
	headers := map[string]string{}
	// inject header into ctx
	for key, value := range req.Header {
		if len(value) > 0 {
			headers[key] = value[0]
		}
	}
	md := metadata.New(headers)
	ctx = metadata.NewIncomingContext(ctx, md)

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")

	if !strings.Contains(contentType, "multipart/form-data") {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
		span.SetStatus(1, "")
		return
	}

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	authUser, err := s.AuthenticateUser(ctx, false)
	if err != nil {
		sta := status.Convert(err)
		switch sta.Code() {
		case codes.NotFound:
			makeJSONResponse(w, 404, "Not found", "User not found")
			span.SetStatus(1, "User not found")
			return
		default:
			makeJSONResponse(w, 401, "Unauthorized", "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			span.SetStatus(1, "Required parameter 'Instill-User-Uid' or 'owner-id' not found in your header")
			return
		}
	}

	path := pathParams["path"]

	ns, modelID, err := s.GetRscNamespaceAndNameID(path)
	if err != nil {
		makeJSONResponse(w, 400, "Model path format error", "Model path format error")
		span.SetStatus(1, "Model path format error")
		return
	}

	pbModel, err := s.GetNamespaceModelByID(req.Context(), ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		makeJSONResponse(w, 404, "Model not found", "The model not found in server")
		span.SetStatus(1, "The model not found in server")
		return
	}

	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
	if err != nil {
		span.SetStatus(1, err.Error())
		return
	}

	modelDef, err := s.GetRepository().GetModelDefinition(modelDefID)
	if err != nil {
		makeJSONResponse(w, 404, "Model definition not found", "The model definition not found in server")
		span.SetStatus(1, "The model definition not found in server")
		return
	}

	usageData := utils.UsageMetricData{
		OwnerUID:           ns.NsUID.String(),
		OwnerType:          mgmtPB.OwnerType_OWNER_TYPE_USER,
		UserUID:            authUser.UID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER,
		ModelUID:           pbModel.Uid,
		TriggerUID:         logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
		ModelDefinitionUID: modelDef.UID.String(),
		ModelTask:          commonPB.Task(pbModel.Task),
	}

	err = req.ParseMultipartForm(4 << 20)
	if err != nil {
		makeJSONResponse(w, 400, "Internal Error", fmt.Sprint("Error while reading file from request %w", err))
		span.SetStatus(1, fmt.Sprint("Error while reading file from request %w", err))
		usageData.Status = mgmtPB.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	var inputInfer interface{}
	var lenInputs = 1
	switch commonPB.Task(pbModel.Task) {
	case commonPB.Task_TASK_CLASSIFICATION,
		commonPB.Task_TASK_DETECTION,
		commonPB.Task_TASK_INSTANCE_SEGMENTATION,
		commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
		commonPB.Task_TASK_OCR,
		commonPB.Task_TASK_KEYPOINT,
		commonPB.Task_TASK_UNSPECIFIED:
		imageInput, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			makeJSONResponse(w, 400, "File Input Error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		lenInputs = len(imageInput)
		inputInfer = imageInput
	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		textToImage, err := parseImageFormDataTextToImageInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		inputInfer = textToImage
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		imageToImage, err := parseImageFormDataImageToImageInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		inputInfer = imageToImage
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnswering, err := parseTextFormDataVisualQuestionAnsweringInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		inputInfer = visualQuestionAnswering
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChat, err := parseTextFormDataTextGenerationChatInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		inputInfer = textGenerationChat
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGeneration, err := parseTextFormDataTextGenerationInputs(req)
		if err != nil {
			makeJSONResponse(w, 400, "Parser input error", err.Error())
			span.SetStatus(1, err.Error())
			usageData.Status = mgmtPB.Status_STATUS_ERRORED
			_ = s.WriteNewDataPoint(ctx, usageData)
			return
		}
		inputInfer = textGeneration
	}
	// check whether model support batching or not. If not, raise an error
	if lenInputs > 1 {
		tritonModelInDB, err := s.GetInferenceEnsembleModel(req.Context(), uuid.FromStringOrNil(pbModel.Uid))
		if tritonModelInDB.Platform == "ensemble" {
			if err != nil {
				makeJSONResponse(w, 404, "Triton Model Error", fmt.Sprintf("The triton model corresponding to model %v do not exist", pbModel.Id))
				span.SetStatus(1, fmt.Sprintf("The triton model corresponding to model %v do not exist", pbModel.Id))
				usageData.Status = mgmtPB.Status_STATUS_ERRORED
				_ = s.WriteNewDataPoint(ctx, usageData)
				return
			}
			configPbFilePath := fmt.Sprintf("%v/%v/config.pbtxt", config.Config.TritonServer.ModelStore, tritonModelInDB.Name)
			doSupportBatch, err := utils.DoSupportBatch(configPbFilePath)
			if err != nil {
				makeJSONResponse(w, 400, "Batching Support Error", err.Error())
				span.SetStatus(1, err.Error())
				usageData.Status = mgmtPB.Status_STATUS_ERRORED
				_ = s.WriteNewDataPoint(ctx, usageData)
				return
			}
			if !doSupportBatch {
				makeJSONResponse(w, 400, "Batching Support Error", "The model do not support batching, so could not make inference with multiple images")
				span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
				usageData.Status = mgmtPB.Status_STATUS_ERRORED
				_ = s.WriteNewDataPoint(ctx, usageData)
				return
			}
		}
	}
	task := commonPB.Task(pbModel.Task)
	var response []*modelPB.TaskOutput
	response, err = s.TriggerNamespaceModelByID(req.Context(), ns, authUser, modelID, inputInfer, task)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
			"Triton inference server",
			"",
			"",
			err.Error(),
		)
		if strings.Contains(err.Error(), "Failed to allocate memory") {
			st, e = sterr.CreateErrorResourceInfo(
				codes.ResourceExhausted,
				"[handler] inference model error",
				"Triton inference server OOM",
				"Out of memory for running the model, maybe try with smaller batch size",
				"",
				err.Error(),
			)
		}

		if e != nil {
			logger.Error(e.Error())
		}
		obj, _ := json.Marshal(st.Details())
		makeJSONResponse(w, 500, st.Message(), string(obj))
		span.SetStatus(1, st.Message())
		usageData.Status = mgmtPB.Status_STATUS_ERRORED
		_ = s.WriteNewDataPoint(ctx, usageData)
		return
	}

	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(200)
	res, err := utils.MarshalOptions.Marshal(&modelPB.TriggerUserModelBinaryFileUploadResponse{
		Task:        task,
		TaskOutputs: response,
	})
	if err != nil {
		makeJSONResponse(w, 500, "Error Predict Model", err.Error())
		span.SetStatus(1, err.Error())
		return
	}

	usageData.Status = mgmtPB.Status_STATUS_COMPLETED
	if err := s.WriteNewDataPoint(ctx, usageData); err != nil {
		logger.Warn("usage and metric data write fail")
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(pbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	_, _ = w.Write(res)

}

func HandleTriggerModelByUpload(s service.Service, w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	inferModelByUpload(s, w, r, pathParams)
}

type GetNamespaceModelCardRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) GetUserModelCard(ctx context.Context, req *modelPB.GetUserModelCardRequest) (resp *modelPB.GetUserModelCardResponse, err error) {
	resp = &modelPB.GetUserModelCardResponse{}
	resp.Readme, err = h.getNamespaceModelCard(ctx, req)

	return resp, err

}

func (h *PublicHandler) GetOrganizationModelCard(ctx context.Context, req *modelPB.GetOrganizationModelCardRequest) (resp *modelPB.GetOrganizationModelCardResponse, err error) {
	resp = &modelPB.GetOrganizationModelCardResponse{}
	resp.Readme, err = h.getNamespaceModelCard(ctx, req)

	return resp, err
}

func (h *PublicHandler) getNamespaceModelCard(ctx context.Context, req GetNamespaceModelCardRequestInterface) (*modelPB.ModelCard, error) {

	eventName := "GetNamespaceModelCard"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, modelID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	dbModel, err := h.service.GetNamespaceModelByID(ctx, ns, authUser, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md", config.Config.TritonServer.ModelStore, authUser.Permalink(), modelID)
	stat, err := os.Stat(readmeFilePath)
	if err != nil { // return empty content base64
		span.SetStatus(1, err.Error())
		return &modelPB.ModelCard{
			Name:     req.GetName(),
			Size:     0,
			Type:     "file",
			Encoding: "base64",
			Content:  []byte(""),
		}, nil
	}

	content, _ := os.ReadFile(readmeFilePath)

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(dbModel),
		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
	)))

	return &modelPB.ModelCard{
		Name:     req.GetName(),
		Size:     int32(stat.Size()),
		Type:     "file",   // currently only support file type
		Encoding: "base64", // currently only support base64 encoding
		Content:  []byte(content),
	}, nil
}

func (h *PublicHandler) GetModelDefinition(ctx context.Context, req *modelPB.GetModelDefinitionRequest) (*modelPB.GetModelDefinitionResponse, error) {

	ctx, span := tracer.Start(ctx, "GetModelDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	definitionID, err := resource.GetDefinitionID(req.Name)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	pbModelDefinition, err := h.service.GetModelDefinition(ctx, definitionID)
	if err != nil {
		return &modelPB.GetModelDefinitionResponse{}, err
	}

	return &modelPB.GetModelDefinitionResponse{ModelDefinition: pbModelDefinition}, nil
}

func (h *PublicHandler) ListModelDefinitions(ctx context.Context, req *modelPB.ListModelDefinitionsRequest) (*modelPB.ListModelDefinitionsResponse, error) {

	ctx, span := tracer.Start(ctx, "ListModelDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	pbModelDefinitions, totalSize, nextPageToken, err := h.service.ListModelDefinitions(ctx, parseView(req.GetView()), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &modelPB.ListModelDefinitionsResponse{}, err
	}

	resp := modelPB.ListModelDefinitionsResponse{
		ModelDefinitions: pbModelDefinitions,
		NextPageToken:    nextPageToken,
		TotalSize:        totalSize,
	}

	return &resp, nil
}
