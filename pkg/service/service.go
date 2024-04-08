package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/model-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

// InferInput is the interface for the input to the model
type InferInput any

// Service is the interface for the service layer
type Service interface {

	// Utils
	AuthenticateUser(ctx context.Context, allowVisitor bool) (authUser *AuthUser, err error)
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient
	GetRepository() repository.Repository
	GetRedisClient() *redis.Client
	GetACLClient() *acl.ACLClient
	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertRepositoryNameToRscName(repositoryName string) (string, error)
	PBToDBModel(ctx context.Context, ns resource.Namespace, pbModel *modelPB.Model) *datamodel.Model
	DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model) (*modelPB.Model, error)
	DBToPBModels(ctx context.Context, dbModels []*datamodel.Model) ([]*modelPB.Model, error)
	DBToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) (*modelPB.ModelDefinition, error)
	DBToPBModelDefinitions(ctx context.Context, dbModelDefinitions []*datamodel.ModelDefinition) ([]*modelPB.ModelDefinition, error)

	// Public
	ListModels(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, showDeleted bool) ([]*modelPB.Model, int32, string, error)
	GetModelByUID(ctx context.Context, authUser *AuthUser, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error)
	ListNamespaceModels(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view modelPB.View, showDeleted bool) ([]*modelPB.Model, int32, string, error)
	GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, view modelPB.View) (*modelPB.Model, error)
	CreateNamespaceModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, model *datamodel.Model) error
	DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string) error
	RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, newModelID string) (*modelPB.Model, error)
	UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, model *modelPB.Model) (*modelPB.Model, error)
	ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, authUser *AuthUser, page int32, pageSize int32, modelID string) ([]*modelPB.ModelVersion, int32, int32, int32, error)
	WatchModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, version string) (*modelPB.State, string, error)

	TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, version string, inferInput InferInput, task commonPB.Task, triggerUID string) ([]*modelPB.TaskOutput, error)
	TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, version string, inferInput InferInput, task commonPB.Task, triggerUID string) (*longrunningpb.Operation, error)

	GetModelDefinition(ctx context.Context, id string) (*modelPB.ModelDefinition, error)
	GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelPB.ModelDefinition, error)
	ListModelDefinitions(ctx context.Context, view modelPB.View, pageSize int32, pageToken string) ([]*modelPB.ModelDefinition, int32, string, error)

	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	// Private
	GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*modelPB.Model, error)
	GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error)
	ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, showDeleted bool) ([]*modelPB.Model, int32, string, error)
	UpdateModelInstanceAdmin(ctx context.Context, ns resource.Namespace, modelID string, hardware string, version string, isDeploy bool) error
	CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error
	DeleteModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error

	// Usage collection
	WriteNewDataPoint(ctx context.Context, data *utils.UsageMetricData) error
}

type service struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	mgmtPublicServiceClient      mgmtPB.MgmtPublicServiceClient
	mgmtPrivateServiceClient     mgmtPB.MgmtPrivateServiceClient
	artifactPrivateServiceClient artifactPB.ArtifactPrivateServiceClient
	temporalClient               client.Client
	ray                          ray.Ray
	aclClient                    *acl.ACLClient
}

// NewService returns a new service instance
func NewService(
	r repository.Repository,
	mp mgmtPB.MgmtPublicServiceClient,
	m mgmtPB.MgmtPrivateServiceClient,
	ar artifactPB.ArtifactPrivateServiceClient,
	rc *redis.Client,
	tc client.Client,
	ra ray.Ray,
	a *acl.ACLClient) Service {
	return &service{
		repository:                   r,
		ray:                          ra,
		mgmtPublicServiceClient:      mp,
		mgmtPrivateServiceClient:     m,
		artifactPrivateServiceClient: ar,
		redisClient:                  rc,
		temporalClient:               tc,
		aclClient:                    a,
	}
}

func InjectAuthUserToContext(ctx context.Context, authUser *AuthUser) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, constant.HeaderAuthTypeKey, authUser.GetACLType())
	ctx = metadata.AppendToOutgoingContext(ctx, constant.HeaderUserUIDKey, authUser.UID.String())
	return ctx
}

type AuthUser struct {
	IsVisitor bool
	UID       uuid.UUID
}

func (a AuthUser) GetACLType() string {
	if a.IsVisitor {
		return "visitor"
	} else {
		return "user"
	}
}

func (a AuthUser) Permalink() string {
	if a.IsVisitor {
		return fmt.Sprintf("visitors/%s", a.UID)
	} else {
		return fmt.Sprintf("users/%s", a.UID)
	}
}

func (s *service) AuthenticateUser(ctx context.Context, allowVisitor bool) (authUser *AuthUser, err error) {
	// Verify if "Instill-User-Uid" is in the header
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType == "user" {
		headerCtxUserUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
		if headerCtxUserUID == "" {
			return nil, ErrUnauthenticated
		}
		return &AuthUser{
			UID:       uuid.FromStringOrNil(headerCtxUserUID),
			IsVisitor: false,
		}, nil
	} else if authType == "visitor" {
		if !allowVisitor {
			return nil, ErrUnauthenticated
		}
		headerCtxVisitorUID := resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
		if headerCtxVisitorUID == "" {
			return nil, ErrUnauthenticated
		}

		return &AuthUser{
			UID:       uuid.FromStringOrNil(headerCtxVisitorUID),
			IsVisitor: true,
		}, nil
	} else {
		return nil, fmt.Errorf("auth type header error")
	}
}

func (s *service) GetRepository() repository.Repository {
	return s.repository
}

// GetRedisClient returns the redis client
func (s *service) GetRedisClient() *redis.Client {
	return s.redisClient
}

// GetACLClient returns the acl client
func (s *service) GetACLClient() *acl.ACLClient {
	return s.aclClient
}

// GetMgmtPrivateServiceClient returns the management private service client
func (s *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return s.mgmtPrivateServiceClient
}

func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	if strings.HasPrefix(name, "users") {
		userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtPB.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(context.Background(), &mgmtPB.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	if strings.HasPrefix(permalink, "users") {
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		userResp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("organizations/%s", userResp.Organization.Id), nil
	}
}

func (s *service) FetchOwnerWithPermalink(permalink string) (*mgmtPB.Owner, error) {
	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}

		return &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}

		return &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}, nil
	}
}

func (s *service) ConvertRepositoryNameToRscName(repositoryName string) (string, error) {
	// repository/flaming-wombat/llava-34b
	splits := strings.Split(repositoryName, "/")
	if len(splits) != 3 {
		return "", status.Errorf(codes.InvalidArgument, "Repository name format error")
	}
	// TODO: how to tell if is user or org
	return fmt.Sprintf("users/%s/models/%s", splits[1], splits[2]), nil
}

func (s *service) GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error) {

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", status.Errorf(codes.InvalidArgument, "Namespace format error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))

	if err != nil {
		return resource.Namespace{}, "", status.Errorf(codes.InvalidArgument, "Namespace format error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsID:   splits[1],
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, status.Errorf(codes.InvalidArgument, "Namespace format error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink((fmt.Sprintf("%s/%s", splits[0], splits[1])))
	if err != nil {
		return resource.Namespace{}, uuid.Nil, status.Errorf(codes.InvalidArgument, "Namespace format error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsID:   splits[1],
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) GetModelByUID(ctx context.Context, authUser *AuthUser, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error) {

	if granted, err := s.aclClient.CheckPermission("model_", modelUID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbModel, err := s.repository.GetModelByUID(ctx, modelUID, view == modelPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.repository.GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel)
}

func (s *service) GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*modelPB.Model, error) {

	dbModel, err := s.repository.GetModelByIDAdmin(ctx, modelID, view == modelPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel)
}

func (s *service) GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error) {

	dbModel, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, view == modelPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel)
}

func (s *service) GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, view modelPB.View) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, view == modelPB.View_VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel)
}

func (s *service) CreateNamespaceModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, model *datamodel.Model) error {

	if ns.NsType == resource.Organization {
		granted, err := s.aclClient.CheckPermission("organization", ns.NsUID, authUser.GetACLType(), authUser.UID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return ErrNoPermission
		}
	} else if ns.NsUID != authUser.UID {
		return ErrNoPermission
	}

	if err := s.repository.CreateNamespaceModel(ctx, model.Owner, model); err != nil {
		return err
	}

	dbCreatedModel, err := s.repository.GetNamespaceModelByID(ctx, model.Owner, model.ID, false)
	if err != nil {
		return err
	}

	nsType, ownerUID, err := resource.GetNamespaceTypeAndUID(model.Owner)
	if err != nil {
		return err
	}
	ownerType := nsType[0 : len(nsType)-1]

	err = s.aclClient.SetOwner("model_", dbCreatedModel.UID, ownerType, ownerUID)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) WatchModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, version string) (*modelPB.State, string, error) {
	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true)
	if err != nil {
		return nil, "", ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, "", err
	} else if !granted {
		return nil, "", ErrNotFound
	}

	name := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)

	state, message, err := s.ray.ModelReady(ctx, name, version)
	if err != nil {
		return nil, "", err
	}

	return state, message, nil
}

func (s *service) TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, version string, inferInput InferInput, task commonPB.Task, triggerUID string) ([]*modelPB.TaskOutput, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	inputJSON, err := json.Marshal(inferInput)
	if err != nil {
		return nil, err
	}

	inputBlobRedisKey := fmt.Sprintf("async_model_request:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		inputBlobRedisKey,
		inputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       triggerUID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerModelWorkflow",
		&worker.TriggerModelWorkflowRequest{
			ModelID:                  dbModel.ID,
			ModelUID:                 dbModel.UID,
			ModelVersion:             version,
			OwnerUID:                 ns.NsUID,
			OwnerType:                string(ns.NsType),
			UserUID:                  userUID,
			UserType:                 mgmtPB.OwnerType_OWNER_TYPE_USER.String(),
			ModelDefinitionUID:       dbModel.ModelDefinitionUID,
			Task:                     task,
			TriggerInputBlobRedisKey: inputBlobRedisKey,
			Mode:                     mgmtPB.Mode_MODE_SYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	var triggerResult *worker.TriggerModelWorkflowResponse
	err = we.Get(ctx, &triggerResult)
	if err != nil {
		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) {
			var details worker.EndUserErrorDetails
			if dErr := applicationErr.Details(&details); dErr == nil && details.Message != "" {
				err = errmsg.AddMessage(err, details.Message)
			}
		}

		return nil, err
	}

	triggerModelResponse := &modelPB.TriggerUserModelResponse{}

	blob, err := s.redisClient.GetDel(ctx, triggerResult.OutputBlobRedisKey).Bytes()
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(blob, triggerModelResponse)
	if err != nil {
		return nil, err
	}

	return triggerModelResponse.TaskOutputs, nil
}

func (s *service) TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, version string, inferInput InferInput, task commonPB.Task, triggerUID string) (*longrunningpb.Operation, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	inputJSON, err := json.Marshal(inferInput)
	if err != nil {
		return nil, err
	}

	inputBlobRedisKey := fmt.Sprintf("async_model_request:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		inputBlobRedisKey,
		inputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       triggerUID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerModelWorkflow",
		&worker.TriggerModelWorkflowRequest{
			ModelID:                  dbModel.ID,
			ModelUID:                 dbModel.UID,
			ModelVersion:             version,
			OwnerUID:                 ns.NsUID,
			OwnerType:                string(ns.NsType),
			UserUID:                  userUID,
			UserType:                 mgmtPB.OwnerType_OWNER_TYPE_USER.String(),
			ModelDefinitionUID:       dbModel.ModelDefinitionUID,
			Task:                     task,
			TriggerInputBlobRedisKey: inputBlobRedisKey,
			Mode:                     mgmtPB.Mode_MODE_ASYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with workflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", triggerUID),
		Done: false,
	}, nil
}

func (s *service) ListModels(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, showDeleted bool) ([]*modelPB.Model, int32, string, error) {

	var uidAllowList []uuid.UUID
	var err error
	role := "reader"

	if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions("model_", authUser.GetACLType(), uuid.Nil, role)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions("model_", authUser.GetACLType(), authUser.UID, role)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions("model_", authUser.GetACLType(), uuid.Nil, role)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions("model_", authUser.GetACLType(), authUser.UID, role)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbModels, totalSize, nextPageToken, err := s.repository.ListModels(ctx, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbModels, err := s.DBToPBModels(ctx, dbModels)
	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) ListNamespaceModels(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view modelPB.View, showDeleted bool) ([]*modelPB.Model, int32, string, error) {

	ownerPermalink := ns.Permalink()

	uidAllowList, err := s.aclClient.ListPermissions("model_", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	dbModels, ps, pt, err := s.repository.ListNamespaceModels(ctx, ownerPermalink, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbModels, err := s.DBToPBModels(ctx, dbModels)
	return pbModels, int32(ps), pt, err
}

func (s *service) ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, authUser *AuthUser, page int32, pageSize int32, modelID string) ([]*modelPB.ModelVersion, int32, int32, int32, error) {
	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true)
	if err != nil {
		return nil, 0, 0, 0, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, 0, 0, 0, err
	} else if !granted {
		return nil, 0, 0, 0, ErrNotFound
	}

	resp, err := s.artifactPrivateServiceClient.ListRepositoryTags(ctx, &artifactPB.ListRepositoryTagsRequest{
		Parent:   fmt.Sprintf("repositories/%s/%s", ns.NsID, modelID),
		Page:     &page,
		PageSize: &pageSize,
	})
	if err != nil {
		return nil, 0, 0, 0, err
	}

	versions := make([]*modelPB.ModelVersion, resp.GetPageSize())

	for _, tag := range resp.GetTags() {
		state, _, err := s.WatchModel(ctx, ns, authUser, modelID, tag.GetId())
		if err != nil {
			return nil, 0, 0, 0, err
		}
		versions = append(versions, &modelPB.ModelVersion{
			Name:       fmt.Sprintf("%s/models/%s/versions/%s", ns.Name(), modelID, tag.GetId()),
			Id:         tag.GetId(),
			Digest:     tag.GetDigest(),
			State:      *state,
			UpdateTime: tag.GetUpdateTime(),
		})
	}

	return versions, resp.GetTotalSize(), resp.GetPageSize(), resp.GetPage(), nil
}

func (s *service) ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, showDeleted bool) ([]*modelPB.Model, int32, string, error) {

	dbModels, totalSize, nextPageToken, err := s.repository.ListModelsAdmin(ctx, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbModels, err := s.DBToPBModels(ctx, dbModels)

	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string) error {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, false)
	if err != nil {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	versions, err := s.repository.ListModelVerions(ctx, dbModel.UID)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if err := s.UpdateModelInstanceAdmin(ctx, ns, dbModel.ID, dbModel.Hardware, version.Version, false); err != nil {
			return err
		}
		if err := s.DeleteModelVersionAdmin(ctx, version); err != nil {
			return err
		}
	}

	err = s.aclClient.Purge("model_", dbModel.UID)
	if err != nil {
		return err
	}

	return s.repository.DeleteNamespaceModelByID(ctx, ownerPermalink, dbModel.ID)
}

func (s *service) RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, newModelID string) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbModel.UID, authUser.GetACLType(), authUser.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if err := s.repository.UpdateNamespaceModelIDByID(ctx, ownerPermalink, modelID, newModelID); err != nil {
		return nil, err
	}

	updatedDBModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, newModelID, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, updatedDBModel)
}

func (s *service) UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, toUpdateModel *modelPB.Model) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbToUpdateModel := s.PBToDBModel(ctx, ns, toUpdateModel)

	if granted, err := s.aclClient.CheckPermission("model_", dbToUpdateModel.UID, authUser.GetACLType(), authUser.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("model_", dbToUpdateModel.UID, authUser.GetACLType(), authUser.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	var err error
	var dbModel *datamodel.Model
	if dbModel, err = s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, false); dbModel == nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespaceModelByID(ctx, ownerPermalink, modelID, dbToUpdateModel); err != nil {
		return nil, err
	}

	updatedDBModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, dbModel.ID, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, updatedDBModel)
}

func (s *service) GetModelDefinition(ctx context.Context, id string) (*modelPB.ModelDefinition, error) {

	dbModelDef, err := s.repository.GetModelDefinition(id)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModelDefinition(ctx, dbModelDef)
}

func (s *service) GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelPB.ModelDefinition, error) {

	dbModelDef, err := s.repository.GetModelDefinitionByUID(uid)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModelDefinition(ctx, dbModelDef)
}

func (s *service) ListModelDefinitions(ctx context.Context, view modelPB.View, pageSize int32, pageToken string) ([]*modelPB.ModelDefinition, int32, string, error) {

	dbModelDefs, nextPageToken, totalSize, err := s.repository.ListModelDefinitions(view, int64(pageSize), pageToken)
	if err != nil {
		return nil, 0, "", err
	}

	pbModelDefs, err := s.DBToPBModelDefinitions(ctx, dbModelDefs)

	return pbModelDefs, int32(totalSize), nextPageToken, err
}

func (s *service) UpdateModelInstanceAdmin(ctx context.Context, ns resource.Namespace, modelID string, hardware string, version string, isDeploy bool) error {

	name := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)
	if err := s.ray.UpdateContainerizedModel(ctx, name, ns.NsID, modelID, version, hardware, isDeploy); err != nil {
		return err
	}

	return nil
}

func (s *service) CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error {
	return s.repository.CreateModelVersionAdmin(ctx, "", version)
}

func (s *service) DeleteModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error {
	return s.repository.DeleteModelVersionAdmin(ctx, "", version)
}
