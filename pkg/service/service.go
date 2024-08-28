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
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/minio"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/model-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
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
	GetACLClient() acl.ACLClientInterface
	GetRayClient() ray.Ray
	GetRscNamespace(ctx context.Context, namespaceID string) (resource.Namespace, error)
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
	CreateNamespaceModel(ctx context.Context, ns resource.Namespace, modelDefinition *datamodel.ModelDefinition, model *modelpb.Model) error
	DeleteNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string) error
	RenameNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, newModelID string) (*modelpb.Model, error)
	UpdateNamespaceModelByID(ctx context.Context, ns resource.Namespace, modelID string, model *modelpb.Model) (*modelpb.Model, error)
	ListNamespaceModelVersions(ctx context.Context, ns resource.Namespace, page int32, pageSize int32, modelID string) ([]*modelpb.ModelVersion, int32, int32, int32, error)
	DeleteModelVersionByID(ctx context.Context, ns resource.Namespace, modelID string, version string) error
	WatchModel(ctx context.Context, ns resource.Namespace, modelID string, version string) (*modelpb.State, string, error)

	TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, reqJSON []byte, task commonpb.Task, triggerUID string) ([]*structpb.Struct, error)
	TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, reqJSON []byte, task commonpb.Task, triggerUID string) (*longrunningpb.Operation, error)

	GetModelDefinition(ctx context.Context, id string) (*modelpb.ModelDefinition, error)
	GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (*modelpb.ModelDefinition, error)
	ListModelDefinitions(ctx context.Context, view modelpb.View, pageSize int32, pageToken string) ([]*modelpb.ModelDefinition, int32, string, error)

	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)
	GetNamespaceModelOperation(ctx context.Context, ns resource.Namespace, modelID string, version string, view modelpb.View) (*longrunningpb.Operation, error)
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

	ListModelTriggers(ctx context.Context, req *modelpb.ListModelRunsRequest, filter filtering.Filter) (*modelpb.ListModelRunsResponse, error)
}

type service struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	mgmtPublicServiceClient      mgmtpb.MgmtPublicServiceClient
	mgmtPrivateServiceClient     mgmtpb.MgmtPrivateServiceClient
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
	temporalClient               client.Client
	ray                          ray.Ray
	aclClient                    acl.ACLClientInterface
	minioClient                  minio.MinioI
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
	a acl.ACLClientInterface,
	minioClient minio.MinioI,
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
		minioClient:                  minioClient,
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
func (s *service) GetACLClient() acl.ACLClientInterface {
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

func (s *service) FetchOwnerWithPermalink(ctx context.Context, permalink string) (*mgmtpb.Owner, error) {
	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := s.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtpb.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	uid := strings.Split(permalink, "/")[1]
	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtpb.LookUpUserAdminRequest{UserUid: uid})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtpb.Owner{Owner: &mgmtpb.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtpb.LookUpOrganizationAdminRequest{OrganizationUid: uid})
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

func (s *service) CreateNamespaceModel(ctx context.Context, ns resource.Namespace, modelDefinition *datamodel.ModelDefinition, model *modelpb.Model) error {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return err
	}

	var modelConfig datamodel.ContainerizedModelConfiguration
	b, err := model.GetConfiguration().MarshalJSON()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}

	bModelConfig, _ := json.Marshal(modelConfig)

	dbModel, err := s.PBToDBModel(ctx, ns, model)
	if err != nil {
		return err
	}
	dbModel.Configuration = bModelConfig
	dbModel.ModelDefinitionUID = modelDefinition.UID

	if err := s.repository.CreateNamespaceModel(ctx, dbModel.Owner, dbModel); err != nil {
		return err
	}

	dbCreatedModel, err := s.repository.GetNamespaceModelByID(ctx, dbModel.Owner, dbModel.ID, false, false)
	if err != nil {
		return err
	}

	nsType, ownerUID, err := resource.GetNamespaceTypeAndUID(dbModel.Owner)
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

	state, message, _, err := s.ray.ModelReady(ctx, name, version)
	if err != nil {
		return nil, "", err
	}

	return state, message, nil
}

// checkRequesterPermission validates that the authenticated user can make
// requests on behalf of the resource identified by the requester UID.
func (s *service) checkRequesterPermission(ctx context.Context, model *datamodel.Model) error {
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType != "user" {
		// Only authenticated users can switch namespaces.
		return ErrUnauthenticated
	}

	requester := resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	authenticatedUser := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if requester == "" || authenticatedUser == requester {
		// Request doesn't contain impersonation.
		return nil
	}

	// The only impersonation that's currently implemented is switching to an
	// organization namespace.
	isMember, err := s.aclClient.CheckPermission(ctx, "organization", uuid.FromStringOrNil(requester), "member")
	if err != nil {
		return errmsg.AddMessage(
			fmt.Errorf("checking organization membership: %w", err),
			"Couldn't check organization membership.",
		)
	}

	if !isMember {
		return fmt.Errorf("authenticated user doesn't belong to requester organization: %w", ErrUnauthenticated)
	}

	// Organizations can only trigger private models owned by themselves.
	// The rest of private models are invisible to them.
	if !model.IsPublic() && model.OwnerUID().String() != requester {
		return fmt.Errorf("model not found: %w", ErrNotFound)
	}

	return nil
}

func (s *service) TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, reqJSON []byte, task commonpb.Task, triggerUID string) ([]*structpb.Struct, error) {

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

	// For now, impersonation is only implemented for model triggers. When
	// this is used in other entrypoints, the requester permission should be
	// checked at a higher level (e.g. handler or middleware).
	if err := s.checkRequesterPermission(ctx, dbModel); err != nil {
		return nil, fmt.Errorf("checking requester permission: %w", err)
	}

	if state, _, numOfActiveReplica, err := s.ray.ModelReady(ctx, fmt.Sprintf("%s/%s", ns.Permalink(), id), version.Version); err != nil {
		return nil, fmt.Errorf("model is not ready to serve requests: %w", err)
	} else if numOfActiveReplica == 0 {
		if *state == modelpb.State_STATE_OFFLINE || *state == modelpb.State_STATE_SCALING_DOWN {
			scalingConfig := ray.GenerateScalingConfig(dbModel.ID)
			name := fmt.Sprintf("%s/%s", ns.Permalink(), dbModel.ID)
			if err := s.ray.UpdateContainerizedModel(ctx, name, ns.NsID, dbModel.ID, version.Version, "", ray.UpScale, scalingConfig); err != nil {
				logger.Warn(fmt.Sprintf("model is not ready to serve requests: %v", err))
			}
		}
		logger.Warn(fmt.Sprintf("model is in %s and has %v active replica, starting new instance now.", state, numOfActiveReplica))
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

	inputReferenceID := minio.GenerateInputRefID()
	// todo: put it in separate workflow activity and store url and file size
	_, _, err = s.minioClient.UploadFileBytes(ctx, inputReferenceID, reqJSON, constant.ContentTypeJSON)
	if err != nil {
		logger.Error("UploadBase64File for input failed", zap.String("inputReferenceID", inputReferenceID), zap.String("reqJSON", string(reqJSON)), zap.Error(err))
		return nil, err
	}

	inputKey := fmt.Sprintf("model_trigger_input_req:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		inputKey,
		reqJSON,
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

	source := datamodel.TriggerSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentEnum, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgent)]
	if ok {
		source = datamodel.TriggerSource(userAgentEnum)
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
			InputKey:           inputKey,
			Mode:               mgmtpb.Mode_MODE_SYNC,
			Hardware:           dbModel.Hardware,
			Visibility:         dbModel.Visibility,
			Source:             source,
			InputReferenceID:   inputReferenceID,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	err = we.Get(ctx, nil)
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

	triggerModelResponse := &modelpb.TriggerNamespaceModelResponse{}

	trigger, err := s.repository.GetModelTriggerByTriggerUID(ctx, triggerUID)
	if err != nil {
		return nil, err
	}

	if !trigger.OutputReferenceID.Valid {
		return nil, fmt.Errorf("trigger output not valid")
	}
	output, err := s.minioClient.GetFile(ctx, trigger.OutputReferenceID.String)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(output, triggerModelResponse)
	if err != nil {
		return nil, err
	}

	return triggerModelResponse.TaskOutputs, nil
}

func (s *service) TriggerAsyncNamespaceModelByID(ctx context.Context, ns resource.Namespace, id string, version *datamodel.ModelVersion, reqJSON []byte, task commonpb.Task, triggerUID string) (*longrunningpb.Operation, error) {

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

	// For now, impersonation is only implemented for model triggers. When
	// this is used in other entrypoints, the requester permission should be
	// checked at a higher level (e.g. handler or middleware).
	if err := s.checkRequesterPermission(ctx, dbModel); err != nil {
		return nil, fmt.Errorf("checking requester permission: %w", err)
	}

	if state, _, numOfActiveReplica, err := s.ray.ModelReady(ctx, fmt.Sprintf("%s/%s", ns.Permalink(), id), version.Version); err != nil {
		return nil, fmt.Errorf("model is not ready to serve requests: %w", err)
	} else if numOfActiveReplica == 0 {
		if *state == modelpb.State_STATE_OFFLINE || *state == modelpb.State_STATE_SCALING_DOWN {
			scalingConfig := ray.GenerateScalingConfig(dbModel.ID)
			name := fmt.Sprintf("%s/%s", ns.Permalink(), dbModel.ID)
			if err := s.ray.UpdateContainerizedModel(ctx, name, ns.NsID, dbModel.ID, version.Version, "", ray.UpScale, scalingConfig); err != nil {
				logger.Warn(fmt.Sprintf("model is not ready to serve requests: %v", err))
			}
		}
		logger.Warn(fmt.Sprintf("model is in %s and has %v active replica, starting new instance now.", state, numOfActiveReplica))
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

	inputReferenceID := minio.GenerateInputRefID()
	// todo: put it in separate workflow activity and store url and file size
	_, _, err = s.minioClient.UploadFileBytes(ctx, inputReferenceID, reqJSON, constant.ContentTypeJSON)
	if err != nil {
		logger.Error("UploadBase64File for input failed", zap.String("inputReferenceID", inputReferenceID), zap.String("parsedInferInput", string(reqJSON)), zap.Error(err))
		return nil, err
	}

	inputKey := fmt.Sprintf("model_trigger_input_req:%s", triggerUID)
	s.redisClient.Set(
		ctx,
		inputKey,
		reqJSON,
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

	source := datamodel.TriggerSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentEnum, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgent)]
	if ok {
		source = datamodel.TriggerSource(userAgentEnum)
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
			InputKey:           inputKey,
			Mode:               mgmtpb.Mode_MODE_ASYNC,
			Hardware:           dbModel.Hardware,
			Visibility:         dbModel.Visibility,
			Source:             source,
			InputReferenceID:   inputReferenceID,
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

	dbModels, totalSize, nextPageToken, err := s.repository.ListModels(ctx, int64(pageSize), pageToken, view == modelpb.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order, visibility)
	if err != nil {
		return nil, 0, "", err
	}
	pbModels, err := s.DBToPBModels(ctx, dbModels, view, true)
	return pbModels, int32(totalSize), nextPageToken, err
}

func (s *service) ListModelTriggers(ctx context.Context, req *modelpb.ListModelRunsRequest, filter filtering.Filter) (*modelpb.ListModelRunsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	logger, _ := custom_logger.GetZapLogger(ctx)

	ns, err := s.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	dbModel, err := s.repository.GetNamespaceModelByID(ctx, ns.Permalink(), req.GetModelId(), true, false)
	if err != nil {
		return nil, err
	}
	ctxUserUID := utils.GetUserUID(ctx)

	isOwner := dbModel.OwnerUID().String() == ctxUserUID

	triggers, totalSize, err := s.repository.ListModelTriggers(ctx, int64(pageSize), int64(page), filter, orderBy, ctxUserUID, isOwner, dbModel.UID.String())
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[string][]byte)
	var referenceIDs []string
	for _, trigger := range triggers {
		if trigger.RequesterUID.String() == ctxUserUID { // only the runner could see their input/output data
			referenceIDs = append(referenceIDs, trigger.InputReferenceID)
			if trigger.OutputReferenceID.Valid {
				referenceIDs = append(referenceIDs, trigger.OutputReferenceID.String)
			}
		}
	}

	logger.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		logger.Error("failed to get files from minio", zap.Error(err))
		return nil, err
	}
	for _, content := range fileContents {
		metadataMap[content.Name] = content.Content
	}

	requesterIDMap := make(map[string]struct{})
	for _, trigger := range triggers {
		requesterIDMap[trigger.RequesterUID.String()] = struct{}{}
	}

	runnerMap := make(map[string]*string)
	for requesterID := range requesterIDMap {
		runner, err := s.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{Uid: requesterID})
		if err != nil {
			return nil, err
		}
		logger.Info("CheckNamespaceByUIDAdmin finished", zap.String("RequesterUID", requesterID), zap.String("runnerId", runner.Id))
		runnerMap[requesterID] = &runner.Id
	}

	pbTriggers := make([]*modelpb.ModelRun, len(triggers))
	for i, trigger := range triggers {
		pbTrigger := &modelpb.ModelRun{
			Uid:        trigger.UID.String(),
			ModelUid:   trigger.ModelUID.String(),
			Version:    trigger.ModelVersion,
			Status:     runpb.RunStatus(trigger.Status),
			Source:     runpb.RunSource(trigger.Source),
			Error:      trigger.Error.Ptr(),
			CreateTime: timestamppb.New(trigger.CreateTime),
			UpdateTime: timestamppb.New(trigger.UpdateTime),
		}

		pbTrigger.RunnerId = runnerMap[trigger.RequesterUID.String()]

		if trigger.TotalDuration.Valid {
			totalDuration := int32(trigger.TotalDuration.Int64)
			pbTrigger.TotalDuration = &totalDuration
		}
		if trigger.EndTime.Valid {
			pbTrigger.EndTime = timestamppb.New(trigger.EndTime.Time)
		}

		if trigger.RequesterUID.String() == ctxUserUID { // only the runner could see their input/output data
			data, ok := metadataMap[trigger.InputReferenceID]
			if !ok {
				return nil, fmt.Errorf("failed to load input metadata. model UID: %s input reference ID: %s", trigger.ModelUID.String(), trigger.InputReferenceID)
			}
			// todo: fix TaskInputs type
			triggerReq := &modelpb.TriggerNamespaceModelRequest{}
			err = protojson.Unmarshal(data, triggerReq)
			if err != nil {
				return nil, err
			}
			pbTrigger.TaskInputs = triggerReq.TaskInputs

			if trigger.OutputReferenceID.Valid {
				data := metadataMap[trigger.OutputReferenceID.String]

				// todo: fix TaskOutputs type
				triggerModelResp := &modelpb.TriggerNamespaceModelResponse{}
				err = protojson.Unmarshal(data, triggerModelResp)
				if err != nil {
					return nil, err
				}

				pbTrigger.TaskOutputs = triggerModelResp.TaskOutputs
			}
		}

		pbTriggers[i] = pbTrigger
	}

	return &modelpb.ListModelRunsResponse{
		Runs:      pbTriggers,
		TotalSize: int32(totalSize),
		PageSize:  pageSize,
		Page:      page,
	}, nil
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

	dbModels, ps, pt, err := s.repository.ListNamespaceModels(ctx, ownerPermalink, int64(pageSize), pageToken, view == modelpb.View_VIEW_BASIC, filter, uidAllowList, showDeleted, order, visibility)
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

			state, _, _, err = s.ray.ModelReady(ctx, fmt.Sprintf("%s/%s", ns.Permalink(), modelID), tag.GetId())
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
	logger, _ := custom_logger.GetZapLogger(ctx)

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
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.NotFound:
				logger.Warn("model version record does not exist in repository_tag table")
			default:
				return err
			}
		} else {
			return err
		}
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
		s.redisClient.Del(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s:%s", userUID, dbModel.UID.String(), version.Version))
	}

	s.redisClient.Del(ctx, fmt.Sprintf("model_trigger_output_key:%s:%s:%s", userUID, dbModel.UID.String(), ""))

	err = s.aclClient.Purge(ctx, "model_", dbModel.UID)
	if err != nil {
		return err
	}

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

	toUpdTags := toUpdateModel.GetTags()

	currentTags := dbModel.TagNames()
	toBeCreatedTagNames := make([]string, 0, len(toUpdTags))
	for _, tag := range toUpdTags {
		if !slices.Contains(currentTags, tag) && !slices.Contains(preserveTags, tag) {
			toBeCreatedTagNames = append(toBeCreatedTagNames, tag)
		}
	}

	toBeDeletedTagNames := make([]string, 0, len(toUpdTags))
	for _, tag := range currentTags {
		if !slices.Contains(toUpdTags, tag) && !slices.Contains(preserveTags, tag) {
			toBeDeletedTagNames = append(toBeDeletedTagNames, tag)
		}
	}
	if len(toBeDeletedTagNames) > 0 {
		err = s.repository.DeleteModelTags(ctx, dbModel.UID, toBeDeletedTagNames)
		if err != nil {
			return nil, err
		}
	}
	if len(toBeCreatedTagNames) > 0 {
		err = s.repository.CreateModelTags(ctx, dbModel.UID, toBeCreatedTagNames)
		if err != nil {
			return nil, err
		}
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

	scalingConfig := ray.GenerateScalingConfig(modelID)

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

func (s *service) pageSizeInRange(pageSize int32) int32 {
	if pageSize <= 0 {
		return repository.DefaultPageSize
	}

	if pageSize > repository.MaxPageSize {
		return repository.MaxPageSize
	}

	return pageSize
}

func (s *service) pageInRange(page int32) int32 {
	if page <= 0 {
		return 0
	}

	return page
}
