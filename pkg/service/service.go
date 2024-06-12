package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
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

// Service is the interface for the service layer
type Service interface {

	// Utils
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient
	GetRepository() repository.Repository
	GetRedisClient() *redis.Client
	GetACLClient() *acl.ACLClient
	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertRepositoryNameToRscName(repositoryName string) (string, error)
	PBToDBModel(ctx context.Context, ns resource.Namespace, pbModel *modelPB.Model) (*datamodel.Model, error)
	DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model, view modelPB.View, checkPermission bool) (*modelPB.Model, error)
	DBToPBModels(ctx context.Context, dbModels []*datamodel.Model, view modelPB.View, checkPermission bool) ([]*modelPB.Model, error)
	DBToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) (*modelPB.ModelDefinition, error)
	DBToPBModelDefinitions(ctx context.Context, dbModelDefinitions []*datamodel.ModelDefinition) ([]*modelPB.ModelDefinition, error)

	// Public
	ListModels(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelPB.Model, int32, string, error)
	GetModelByUID(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error)
	ListNamespaceModels(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelPB.Model, int32, string, error)
	GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, view modelPB.View) (*modelPB.Model, error)
	CreateNamespaceModel(ctx context.Context, ns resource.Namespace, model *datamodel.Model) error
	DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string) error
	RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, newModelID string) (*modelPB.Model, error)
	UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, model *modelPB.Model) (*modelPB.Model, error)
	ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, page int32, pageSize int32, modelID string) ([]*modelPB.ModelVersion, int32, int32, int32, error)
	WatchModel(ctx context.Context, ns resource.Namespace, modelID string, version string) (*modelPB.State, string, error)

	TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, parsedInferInput []byte, task commonPB.Task, triggerUID string) ([]*modelPB.TaskOutput, error)
	TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, inferInput []byte, parsedInferInput []byte, task commonPB.Task, triggerUID string) (*longrunningpb.Operation, error)

	GetModelDefinition(ctx context.Context, id string) (*modelPB.ModelDefinition, error)
	GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelPB.ModelDefinition, error)
	ListModelDefinitions(ctx context.Context, view modelPB.View, pageSize int32, pageToken string) ([]*modelPB.ModelDefinition, int32, string, error)

	CreateModelPrediction(ctx context.Context, prediction *datamodel.ModelPrediction) error

	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)
	GetNamespaceLatestModelOperation(ctx context.Context, ns resource.Namespace, modelID string, view modelPB.View) (*longrunningpb.Operation, error)

	// Private
	GetModelByIDAdmin(ctx context.Context, ns resource.Namespace, modelID string, view modelPB.View) (*modelPB.Model, error)
	GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error)
	ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, filter filtering.Filter, showDeleted bool) ([]*modelPB.Model, int32, string, error)
	UpdateModelInstanceAdmin(ctx context.Context, ns resource.Namespace, modelID string, hardware string, version string, isDeploy bool) error
	CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error
	GetModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, versionID string) (*datamodel.ModelVersion, error)
	DeleteModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, versionID string) error

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
	instillCoreHost              string
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
	a *acl.ACLClient,
	h string) Service {
	return &service{
		repository:                   r,
		ray:                          ra,
		mgmtPublicServiceClient:      mp,
		mgmtPrivateServiceClient:     m,
		artifactPrivateServiceClient: ar,
		redisClient:                  rc,
		temporalClient:               tc,
		aclClient:                    a,
		instillCoreHost:              h,
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

func (s *service) FetchOwnerWithPermalink(ctx context.Context, permalink string) (*mgmtPB.Owner, error) {
	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := s.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtPB.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil

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

func (s *service) GetModelByUID(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error) {

	dbModel, err := s.repository.GetModelByUID(ctx, modelUID, view == modelPB.View_VIEW_BASIC, false)
	if err != nil {
		return nil, err
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", modelUID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	modelDef, err := s.repository.GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, true)
}

func (s *service) GetModelByIDAdmin(ctx context.Context, ns resource.Namespace, modelID string, view modelPB.View) (*modelPB.Model, error) {

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ns.Permalink(), modelID, view == modelPB.View_VIEW_BASIC, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, false)
}

func (s *service) GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*modelPB.Model, error) {

	dbModel, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, view == modelPB.View_VIEW_BASIC, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, false)
}

func (s *service) GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, view modelPB.View) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, view == modelPB.View_VIEW_BASIC, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, true)
}

func (s *service) CreateNamespaceModel(ctx context.Context, ns resource.Namespace, model *datamodel.Model) error {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return err
	}

	if err := s.repository.CreateNamespaceModel(ctx, model.Owner, model); err != nil {
		return err
	}

	dbCreatedModel, err := s.repository.GetNamespaceModelByID(ctx, model.Owner, model.ID, false, false)
	if err != nil {
		return err
	}

	nsType, ownerUID, err := resource.GetNamespaceTypeAndUID(model.Owner)
	if err != nil {
		return err
	}
	ownerType := nsType[0 : len(nsType)-1]

	if err := s.aclClient.SetOwner(ctx, "model_", dbCreatedModel.UID, ownerType, ownerUID); err != nil {
		return err
	}

	if dbCreatedModel.Visibility == datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC) {
		if err := s.aclClient.SetPublicModelPermission(ctx, dbCreatedModel.UID); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) WatchModel(ctx context.Context, ns resource.Namespace, modelID string, version string) (*modelPB.State, string, error) {
	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true, false)
	if err != nil {
		return nil, "", ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
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

func (s *service) TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, parsedInferInput []byte, task commonPB.Task, triggerUID string) ([]*modelPB.TaskOutput, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, id, false, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	parsedInputKey := fmt.Sprintf("model_trigger_input_parsed:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		parsedInputKey,
		parsedInferInput,
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
			ModelID:            dbModel.ID,
			ModelUID:           dbModel.UID,
			ModelVersion:       *version,
			OwnerUID:           ns.NsUID,
			OwnerType:          string(ns.NsType),
			UserUID:            userUID,
			UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER.String(),
			ModelDefinitionUID: dbModel.ModelDefinitionUID,
			Task:               task,
			ParsedInputKey:     parsedInputKey,
			Mode:               mgmtPB.Mode_MODE_SYNC,
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

	blob, err := s.redisClient.GetDel(ctx, triggerResult.OutputKey).Bytes()
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(blob, triggerModelResponse)
	if err != nil {
		return nil, err
	}

	return triggerModelResponse.TaskOutputs, nil
}

func (s *service) TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, inferInput []byte, parsedInferInput []byte, task commonPB.Task, triggerUID string) (*longrunningpb.Operation, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, id, false, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	inputKey := fmt.Sprintf("model_trigger_input:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		inputKey,
		inferInput,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	parsedInputKey := fmt.Sprintf("model_trigger_input_parsed:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		parsedInputKey,
		parsedInferInput,
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
			TriggerUID:         uuid.FromStringOrNil(triggerUID),
			ModelID:            dbModel.ID,
			ModelUID:           dbModel.UID,
			ModelVersion:       *version,
			OwnerUID:           ns.NsUID,
			OwnerType:          string(ns.NsType),
			UserUID:            userUID,
			UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER.String(),
			ModelDefinitionUID: dbModel.ModelDefinitionUID,
			Task:               task,
			InputKey:           inputKey,
			ParsedInputKey:     parsedInputKey,
			Mode:               mgmtPB.Mode_MODE_ASYNC,
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

func (s *service) ListModels(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelPB.Model, int32, string, error) {

	var uidAllowList []uuid.UUID
	var err error
	role := "reader"

	if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "model_", role, false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbModels, totalSize, nextPageToken, err := s.repository.ListModels(ctx, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order)
	if err != nil {
		return nil, 0, "", err
	}
	pbModels, err := s.DBToPBModels(ctx, dbModels, view, true)
	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) ListNamespaceModels(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view modelPB.View, visibility *modelPB.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelPB.Model, int32, string, error) {

	ownerPermalink := ns.Permalink()
	var uidAllowList []uuid.UUID
	var err error
	role := "reader"

	if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == modelPB.Model_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "model_", role, false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbModels, ps, pt, err := s.repository.ListNamespaceModels(ctx, ownerPermalink, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order)
	if err != nil {
		return nil, 0, "", err
	}

	pbModels, err := s.DBToPBModels(ctx, dbModels, view, true)
	return pbModels, int32(ps), pt, err
}

func (s *service) ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, page int32, pageSize int32, modelID string) ([]*modelPB.ModelVersion, int32, int32, int32, error) {
	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true, false)
	if err != nil {
		return nil, 0, 0, 0, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
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

	tags := resp.GetTags()

	versions := make([]*modelPB.ModelVersion, len(tags))

	for i, tag := range tags {
		state, _, err := s.ray.ModelReady(ctx, fmt.Sprintf("%s/%s", ns.Permalink(), modelID), tag.GetId())
		if err != nil {
			state = modelPB.State_STATE_ERROR.Enum()
		}
		versions[i] = &modelPB.ModelVersion{
			Name:       fmt.Sprintf("%s/models/%s/versions/%s", ns.Name(), modelID, tag.GetId()),
			Id:         tag.GetId(),
			Digest:     tag.GetDigest(),
			State:      *state,
			UpdateTime: tag.GetUpdateTime(),
		}
	}

	return versions, resp.GetTotalSize(), resp.GetPageSize(), resp.GetPage(), nil
}

func (s *service) ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelPB.View, filter filtering.Filter, showDeleted bool) ([]*modelPB.Model, int32, string, error) {

	dbModels, totalSize, nextPageToken, err := s.repository.ListModelsAdmin(ctx, int64(pageSize), pageToken, view == modelPB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbModels, err := s.DBToPBModels(ctx, dbModels, view, false)

	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string) error {

	ownerPermalink := ns.Permalink()

	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, false, false)
	if err != nil {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	versions, err := s.repository.ListModelVersions(ctx, dbModel.UID)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if err := s.UpdateModelInstanceAdmin(ctx, ns, dbModel.ID, dbModel.Hardware, version.Version, false); err != nil {
			return err
		}
		if err := s.DeleteModelVersionAdmin(ctx, dbModel.UID, version.Version); err != nil {
			return err
		}
	}

	err = s.aclClient.Purge(ctx, "model_", dbModel.UID)
	if err != nil {
		return err
	}

	s.redisClient.Del(ctx, fmt.Sprintf("model_trigger_input:%s:%s", userUID, dbModel.UID.String()))

	return s.repository.DeleteNamespaceModelByID(ctx, ownerPermalink, dbModel.ID)
}

func (s *service) RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, newModelID string) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if err := s.repository.UpdateNamespaceModelIDByID(ctx, ownerPermalink, modelID, newModelID); err != nil {
		return nil, err
	}

	updatedDBModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, newModelID, false, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, updatedDBModel, modelPB.View_VIEW_BASIC, true)
}

func (s *service) UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, toUpdateModel *modelPB.Model) (*modelPB.Model, error) {

	ownerPermalink := ns.Permalink()

	dbToUpdateModel, err := s.PBToDBModel(ctx, ns, toUpdateModel)
	if err != nil {
		return nil, err
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbToUpdateModel.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "model_", dbToUpdateModel.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	var dbModel *datamodel.Model
	if dbModel, err = s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, false, false); dbModel == nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespaceModelByID(ctx, ownerPermalink, modelID, dbToUpdateModel); err != nil {
		return nil, err
	}

	updatedDBModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, dbModel.ID, false, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	if updatedDBModel.Visibility == datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC) {
		if err := s.aclClient.SetPublicModelPermission(ctx, updatedDBModel.UID); err != nil {
			return nil, err
		}
	} else if updatedDBModel.Visibility == datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE) {
		if err := s.aclClient.DeletePublicModelPermission(ctx, updatedDBModel.UID); err != nil {
			return nil, err
		}
	}

	if updatedDBModel.Hardware != dbModel.Hardware {
		versions, totalSize, _, page, err := s.ListNamespaceModelVersions(ctx, ns, 0, 10, updatedDBModel.ID)
		if err != nil {
			return nil, err
		}
		for len(versions) < int(totalSize) {
			page += 1
			v, _, _, _, err := s.ListNamespaceModelVersions(ctx, ns, page, 10, updatedDBModel.ID)
			if err != nil {
				return nil, err
			}
			versions = append(versions, v...)
		}

		for _, v := range versions {
			if err := s.UpdateModelInstanceAdmin(ctx, ns, updatedDBModel.ID, "", v.Id, false); err != nil {
				return nil, err
			}
			if err := s.UpdateModelInstanceAdmin(ctx, ns, updatedDBModel.ID, updatedDBModel.Hardware, v.Id, true); err != nil {
				return nil, err
			}
		}
	}

	return s.DBToPBModel(ctx, modelDef, updatedDBModel, modelPB.View_VIEW_BASIC, true)
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

func (s *service) GetModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, versionID string) (*datamodel.ModelVersion, error) {
	return s.repository.GetModelVersionByID(ctx, modelUID, versionID)
}

func (s *service) CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error {
	return s.repository.CreateModelVersion(ctx, "", version)
}

func (s *service) DeleteModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, versionID string) error {
	return s.repository.DeleteModelVersionByID(ctx, modelUID, versionID)
}

func (s *service) CreateModelPrediction(ctx context.Context, prediction *datamodel.ModelPrediction) error {
	return s.repository.CreateModelPrediction(ctx, prediction)
}
