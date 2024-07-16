package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"

	"github.com/instill-ai/x/errmsg"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/model-backend/pkg/worker"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

// Service is the interface for the service layer
type Service interface {

	// Utils
	GetMgmtPrivateServiceClient() mgmtpb.MgmtPrivateServiceClient
	GetArtifactPrivateServiceClient() artifactpb.ArtifactPrivateServiceClient
	GetRepository() repository.Repository
	GetRedisClient() *redis.Client
	GetACLClient() *acl.ACLClient
	GetRayClient() ray.Ray
	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertRepositoryNameToRscName(repositoryName string) (string, error)
	PBToDBModel(ctx context.Context, ns resource.Namespace, pbModel *modelpb.Model) (*datamodel.Model, error)
	DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model, view modelpb.View, checkPermission bool) (*modelpb.Model, error)
	DBToPBModels(ctx context.Context, dbModels []*datamodel.Model, view modelpb.View, checkPermission bool) ([]*modelpb.Model, error)
	DBToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) (*modelpb.ModelDefinition, error)
	DBToPBModelDefinitions(ctx context.Context, dbModelDefinitions []*datamodel.ModelDefinition) ([]*modelpb.ModelDefinition, error)

	// Public
	ListModels(ctx context.Context, pageSize int32, pageToken string, view modelpb.View, visibility *modelpb.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelpb.Model, int32, string, error)
	GetModelByUID(ctx context.Context, modelUID uuid.UUID, view modelpb.View) (*modelpb.Model, error)
	ListNamespaceModels(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view modelpb.View, visibility *modelpb.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelpb.Model, int32, string, error)
	GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*modelpb.Model, error)
	CreateNamespaceModel(ctx context.Context, ns resource.Namespace, model *datamodel.Model) error
	DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string) error
	RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, newModelID string) (*modelpb.Model, error)
	UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, model *modelpb.Model) (*modelpb.Model, error)
	ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, page int32, pageSize int32, modelID string) ([]*modelpb.ModelVersion, int32, int32, int32, error)
	DeleteModelVersionByID(ctx context.Context, ns resource.Namespace, modelID string, version string) error
	WatchModel(ctx context.Context, ns resource.Namespace, modelID string, version string) (*modelpb.State, string, error)

	TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, parsedInferInput []byte, task commonpb.Task, triggerUID string) ([]*modelpb.TaskOutput, error)
	TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, inferInput []byte, parsedInferInput []byte, task commonpb.Task, triggerUID string) (*longrunningpb.Operation, error)

	GetModelDefinition(ctx context.Context, id string) (*modelpb.ModelDefinition, error)
	GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelpb.ModelDefinition, error)
	ListModelDefinitions(ctx context.Context, view modelpb.View, pageSize int32, pageToken string) ([]*modelpb.ModelDefinition, int32, string, error)

	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)
	GetNamespaceLatestModelOperation(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*longrunningpb.Operation, error)

	// Private
	GetModelByIDAdmin(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*modelpb.Model, error)
	GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelpb.View) (*modelpb.Model, error)
	ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelpb.View, filter filtering.Filter, showDeleted bool) ([]*modelpb.Model, int32, string, error)
	UpdateModelInstanceAdmin(ctx context.Context, ns resource.Namespace, modelID string, hardware string, version string, action ray.Action) error
	CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error
	GetModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, version string) (*datamodel.ModelVersion, error)

	// Usage collection
	WriteNewDataPoint(ctx context.Context, data *utils.UsageMetricData) error
}

type service struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	mgmtPublicServiceClient      mgmtpb.MgmtPublicServiceClient
	mgmtPrivateServiceClient     mgmtpb.MgmtPrivateServiceClient
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
	temporalClient               client.Client
	ray                          ray.Ray
	aclClient                    *acl.ACLClient
	instillCoreHost              string
}

// NewService returns a new service instance
func NewService(
	r repository.Repository,
	mp mgmtpb.MgmtPublicServiceClient,
	m mgmtpb.MgmtPrivateServiceClient,
	ar artifactpb.ArtifactPrivateServiceClient,
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

func (s *service) generateScalingConfig(modelID string) []string {
	if strings.HasPrefix(modelID, DummyModelPrefix) {
		return []string{
			fmt.Sprintf("-e %s=%v", ray.EnvIsTestModel, "true"),
		}
	}

	return []string{}
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
func (s *service) GetMgmtPrivateServiceClient() mgmtpb.MgmtPrivateServiceClient {
	return s.mgmtPrivateServiceClient
}

// GetArtifactPrivateServiceClient returns the management private service client
func (s *service) GetArtifactPrivateServiceClient() artifactpb.ArtifactPrivateServiceClient {
	return s.artifactPrivateServiceClient
}

// GetRayClient returns the ray client
func (s *service) GetRayClient() ray.Ray {
	return s.ray
}

func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	if strings.HasPrefix(name, "users") {
		userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtpb.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(context.Background(), &mgmtpb.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	if strings.HasPrefix(permalink, "users") {
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtpb.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		userResp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtpb.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("organizations/%s", userResp.Organization.Id), nil
	}
}

func (s *service) FetchOwnerWithPermalink(ctx context.Context, permalink string) (*mgmtpb.Owner, error) {
	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := s.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtpb.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtpb.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtpb.Owner{Owner: &mgmtpb.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtpb.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtpb.Owner{Owner: &mgmtpb.Owner_Organization{Organization: resp.Organization}}
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

func (s *service) GetModelByUID(ctx context.Context, modelUID uuid.UUID, view modelpb.View) (*modelpb.Model, error) {

	dbModel, err := s.repository.GetModelByUID(ctx, modelUID, view == modelpb.View_VIEW_BASIC, false)
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

func (s *service) GetModelByIDAdmin(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*modelpb.Model, error) {

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ns.Permalink(), modelID, view == modelpb.View_VIEW_BASIC, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, false)
}

func (s *service) GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelpb.View) (*modelpb.Model, error) {

	dbModel, err := s.repository.GetModelByUIDAdmin(ctx, modelUID, view == modelpb.View_VIEW_BASIC, false)
	if err != nil {
		return nil, err
	}

	modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModel(ctx, modelDef, dbModel, view, false)
}

func (s *service) GetNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, view modelpb.View) (*modelpb.Model, error) {

	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, view == modelpb.View_VIEW_BASIC, false)
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

	if dbCreatedModel.Visibility == datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PUBLIC) {
		if err := s.aclClient.SetPublicModelPermission(ctx, dbCreatedModel.UID); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) WatchModel(ctx context.Context, ns resource.Namespace, modelID string, version string) (*modelpb.State, string, error) {
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

	_, err = s.GetModelVersionAdmin(ctx, dbModel.UID, version)
	if err != nil {
		return modelpb.State_STATE_ERROR.Enum(), "", status.New(codes.NotFound, "Model version not found").Err()
	}

	name := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)

	state, message, err := s.ray.ModelReady(ctx, name, version)
	if err != nil {
		return nil, "", err
	}

	return state, message, nil
}

func (s *service) TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, parsedInferInput []byte, task commonpb.Task, triggerUID string) ([]*modelpb.TaskOutput, error) {

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
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

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
			UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER.String(),
			RequesterUID:       requesterUID,
			ModelDefinitionUID: dbModel.ModelDefinitionUID,
			Task:               task,
			ParsedInputKey:     parsedInputKey,
			Mode:               mgmtpb.Mode_MODE_SYNC,
			Hardware:           dbModel.Hardware,
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

	triggerModelResponse := &modelpb.TriggerUserModelResponse{}

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

func (s *service) TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, inferInput []byte, parsedInferInput []byte, task commonpb.Task, triggerUID string) (*longrunningpb.Operation, error) {

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
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

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
			UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER.String(),
			RequesterUID:       requesterUID,
			ModelDefinitionUID: dbModel.ModelDefinitionUID,
			Task:               task,
			ParsedInputKey:     parsedInputKey,
			Mode:               mgmtpb.Mode_MODE_ASYNC,
			Hardware:           dbModel.Hardware,
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

func (s *service) ListModels(ctx context.Context, pageSize int32, pageToken string, view modelpb.View, visibility *modelpb.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelpb.Model, int32, string, error) {

	var uidAllowList []uuid.UUID
	var err error
	role := "reader"

	if visibility != nil && *visibility == modelpb.Model_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == modelpb.Model_VISIBILITY_PRIVATE {
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

	dbModels, totalSize, nextPageToken, err := s.repository.ListModels(ctx, int64(pageSize), pageToken, view == modelpb.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order)
	if err != nil {
		return nil, 0, "", err
	}
	pbModels, err := s.DBToPBModels(ctx, dbModels, view, true)
	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) ListNamespaceModels(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view modelpb.View, visibility *modelpb.Model_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*modelpb.Model, int32, string, error) {

	ownerPermalink := ns.Permalink()
	var uidAllowList []uuid.UUID
	var err error
	role := "reader"

	if visibility != nil && *visibility == modelpb.Model_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "model_", role, true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == modelpb.Model_VISIBILITY_PRIVATE {
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

	dbModels, ps, pt, err := s.repository.ListNamespaceModels(ctx, ownerPermalink, int64(pageSize), pageToken, view == modelpb.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order)
	if err != nil {
		return nil, 0, "", err
	}

	pbModels, err := s.DBToPBModels(ctx, dbModels, view, true)
	return pbModels, int32(ps), pt, err
}

func (s *service) ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, page int32, pageSize int32, modelID string) ([]*modelpb.ModelVersion, int32, int32, int32, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

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

	resp, err := s.artifactPrivateServiceClient.ListRepositoryTags(ctx, &artifactpb.ListRepositoryTagsRequest{
		Parent:   fmt.Sprintf("repositories/%s/%s", ns.NsID, modelID),
		Page:     &page,
		PageSize: &pageSize,
	})
	if err != nil {
		return nil, 0, 0, 0, err
	}

	tags := resp.GetTags()

	versions := []*modelpb.ModelVersion{}

	for _, tag := range tags {
		var state *modelpb.State
		var updateTime time.Time
		dbVersion, err := s.GetModelVersionAdmin(ctx, dbModel.UID, tag.GetId())
		if err != nil {
			continue
		} else {

			if dbVersion.Digest == "" {
				if err := s.repository.UpdateModelVersionDigestByID(ctx, dbModel.UID, tag.GetId(), tag.GetDigest()); err != nil {
					logger.Warn(fmt.Sprintf("Upsert missing image digest err: %v", err))
				}
			}

			state, _, err = s.ray.ModelReady(ctx, fmt.Sprintf("%s/%s", ns.Permalink(), modelID), tag.GetId())
			if err != nil {
				state = modelpb.State_STATE_ERROR.Enum()
			}
			updateTime = dbVersion.UpdateTime
		}
		versions = append(versions, &modelpb.ModelVersion{
			Name:       fmt.Sprintf("%s/models/%s/versions/%s", ns.Name(), modelID, tag.GetId()),
			Version:    tag.GetId(),
			Digest:     tag.GetDigest(),
			State:      *state,
			UpdateTime: timestamppb.New(updateTime),
		})
	}

	return versions, resp.GetTotalSize(), resp.GetPageSize(), resp.GetPage(), nil
}

func (s *service) DeleteModelVersionByID(ctx context.Context, ns resource.Namespace, modelID string, version string) error {
	ownerPermalink := ns.Permalink()

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ownerPermalink, modelID, true, false)
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

	dbVersion, err := s.repository.GetModelVersionByID(ctx, dbModel.UID, version)
	if err != nil {
		return err
	}

	dbVersions, err := s.repository.ListModelVersionsByDigest(ctx, dbModel.UID, dbVersion.Digest)
	if err != nil {
		return err
	}

	if _, err := s.artifactPrivateServiceClient.DeleteRepositoryTag(ctx, &artifactpb.DeleteRepositoryTagRequest{
		Name: fmt.Sprintf("repositories/%s/%s/tags/%s", ns.NsID, modelID, version),
	}); err != nil {
		return err
	}

	if err := s.repository.DeleteModelVersionByDigest(ctx, dbModel.UID, dbVersion.Digest); err != nil {
		return err
	}

	for _, v := range dbVersions {
		if err := s.UpdateModelInstanceAdmin(ctx, ns, modelID, dbModel.Hardware, v.Version, ray.Undeploy); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) ListModelsAdmin(ctx context.Context, pageSize int32, pageToken string, view modelpb.View, filter filtering.Filter, showDeleted bool) ([]*modelpb.Model, int32, string, error) {

	dbModels, totalSize, nextPageToken, err := s.repository.ListModelsAdmin(ctx, int64(pageSize), pageToken, view == modelpb.View_VIEW_BASIC, filter, showDeleted)
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

	versions, err := s.repository.ListModelVersions(ctx, dbModel.UID, true)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if err := s.UpdateModelInstanceAdmin(ctx, ns, dbModel.ID, dbModel.Hardware, version.Version, ray.Undeploy); err != nil {
			return err
		}
		if err := s.DeleteModelVersionByID(ctx, ns, modelID, version.Version); err != nil {
			return err
		}
	}

	err = s.aclClient.Purge(ctx, "model_", dbModel.UID)
	if err != nil {
		return err
	}

	s.redisClient.Del(ctx, fmt.Sprintf("model_trigger_input:%s:%s", userUID, dbModel.UID.String()))
	s.redisClient.Del(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s", userUID, dbModel.UID.String()))

	return s.repository.DeleteNamespaceModelByID(ctx, ownerPermalink, dbModel.ID)
}

func (s *service) RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, newModelID string) (*modelpb.Model, error) {

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

	return s.DBToPBModel(ctx, modelDef, updatedDBModel, modelpb.View_VIEW_BASIC, true)
}

func (s *service) UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, toUpdateModel *modelpb.Model) (*modelpb.Model, error) {

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

	if updatedDBModel.Visibility == datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PUBLIC) {
		if err := s.aclClient.SetPublicModelPermission(ctx, updatedDBModel.UID); err != nil {
			return nil, err
		}
	} else if updatedDBModel.Visibility == datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PRIVATE) {
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
			if err := s.UpdateModelInstanceAdmin(ctx, ns, updatedDBModel.ID, "", v.Version, ray.Undeploy); err != nil {
				return nil, err
			}
			if err := s.UpdateModelInstanceAdmin(ctx, ns, updatedDBModel.ID, updatedDBModel.Hardware, v.Version, ray.Deploy); err != nil {
				return nil, err
			}
		}
	}

	return s.DBToPBModel(ctx, modelDef, updatedDBModel, modelpb.View_VIEW_BASIC, true)
}

func (s *service) GetModelDefinition(ctx context.Context, id string) (*modelpb.ModelDefinition, error) {

	dbModelDef, err := s.repository.GetModelDefinition(id)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModelDefinition(ctx, dbModelDef)
}

func (s *service) GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelpb.ModelDefinition, error) {

	dbModelDef, err := s.repository.GetModelDefinitionByUID(uid)
	if err != nil {
		return nil, err
	}

	return s.DBToPBModelDefinition(ctx, dbModelDef)
}

func (s *service) ListModelDefinitions(ctx context.Context, view modelpb.View, pageSize int32, pageToken string) ([]*modelpb.ModelDefinition, int32, string, error) {

	dbModelDefs, nextPageToken, totalSize, err := s.repository.ListModelDefinitions(view, int64(pageSize), pageToken)
	if err != nil {
		return nil, 0, "", err
	}

	pbModelDefs, err := s.DBToPBModelDefinitions(ctx, dbModelDefs)

	return pbModelDefs, int32(totalSize), nextPageToken, err
}

func (s *service) UpdateModelInstanceAdmin(ctx context.Context, ns resource.Namespace, modelID string, hardware string, version string, action ray.Action) error {

	scalingConfig := s.generateScalingConfig(modelID)

	name := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)
	if err := s.ray.UpdateContainerizedModel(ctx, name, ns.NsID, modelID, version, hardware, action, scalingConfig); err != nil {
		return err
	}

	return nil
}

func (s *service) GetModelVersionAdmin(ctx context.Context, modelUID uuid.UUID, version string) (*datamodel.ModelVersion, error) {
	return s.repository.GetModelVersionByID(ctx, modelUID, version)
}

func (s *service) CreateModelVersionAdmin(ctx context.Context, version *datamodel.ModelVersion) error {
	return s.repository.CreateModelVersion(ctx, "", version)
}
