package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/utils"

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
	WatchModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, versionTag string) (*modelPB.State, error)

	TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error)

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
	repository               repository.Repository
	redisClient              *redis.Client
	mgmtPublicServiceClient  mgmtPB.MgmtPublicServiceClient
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	temporalClient           client.Client
	ray                      ray.Ray
	aclClient                *acl.ACLClient
}

// NewService returns a new service instance
func NewService(
	r repository.Repository,
	mp mgmtPB.MgmtPublicServiceClient,
	m mgmtPB.MgmtPrivateServiceClient,
	rc *redis.Client,
	tc client.Client,
	ra ray.Ray,
	a *acl.ACLClient) Service {
	return &service{
		repository:               r,
		ray:                      ra,
		mgmtPublicServiceClient:  mp,
		mgmtPrivateServiceClient: m,
		redisClient:              rc,
		temporalClient:           tc,
		aclClient:                a,
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
	} else {
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

func (s *service) WatchModel(ctx context.Context, ns resource.Namespace, authUser *AuthUser, modelID string, versionTag string) (*modelPB.State, error) {
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

	name := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)

	// TODO: implement model instance
	state, err := s.ray.ModelReady(ctx, name, versionTag)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (s *service) TriggerNamespaceModelByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error) {

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

	name := fmt.Sprintf("%s/%s", ns.Permalink(), dbModel.ID)

	var postprocessResponse any
	modelMetadataResponse := s.ray.ModelMetadataRequest(ctx, name, "default")
	if modelMetadataResponse == nil {
		return nil, fmt.Errorf("model is offline")
	}
	inferResponse, err := s.ray.ModelInferRequest(ctx, task, inferInput, name, "default", modelMetadataResponse)
	if err != nil {
		return nil, err
	}
	postprocessResponse, err = ray.PostProcess(inferResponse, modelMetadataResponse, task)
	if err != nil {
		return nil, err
	}

	switch task {
	case commonPB.Task_TASK_CLASSIFICATION:
		clsResponses := postprocessResponse.([]string)
		var clsOutputs []*modelPB.TaskOutput
		for _, clsRes := range clsResponses {
			clsResSplit := strings.Split(clsRes, ":")
			if len(clsResSplit) == 2 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.TaskOutput{
					Output: &modelPB.TaskOutput_Classification{
						Classification: &modelPB.ClassificationOutput{
							Category: clsResSplit[1],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else if len(clsResSplit) == 3 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.TaskOutput{
					Output: &modelPB.TaskOutput_Classification{
						Classification: &modelPB.ClassificationOutput{
							Category: clsResSplit[2],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else {
				return nil, fmt.Errorf("unable to decode inference output")
			}
		}
		if len(clsOutputs) == 0 {
			clsOutputs = append(clsOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Classification{
					Classification: &modelPB.ClassificationOutput{},
				},
			})
		}
		return clsOutputs, nil
	case commonPB.Task_TASK_DETECTION:
		detResponses := postprocessResponse.(ray.DetectionOutput)
		batchedOutputDataBboxes := detResponses.Boxes
		batchedOutputDataLabels := detResponses.Labels
		var detOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var detOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Detection{
					Detection: &modelPB.DetectionOutput{
						Objects: []*modelPB.DetectionObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Ray to be able to batch Tensors
				if label != "0" {
					bbObj := &modelPB.DetectionObject{
						Category: label,
						Score:    box[4],
						// Convert x1y1x2y2 to xywh where xy is top-left corner
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2] - box[0],
							Height: box[3] - box[1],
						},
					}
					detOutput.GetDetection().Objects = append(detOutput.GetDetection().Objects, bbObj)
				}
			}
			detOutputs = append(detOutputs, &detOutput)
		}
		if len(detOutputs) == 0 {
			detOutputs = append(detOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Detection{
					Detection: &modelPB.DetectionOutput{
						Objects: []*modelPB.DetectionObject{},
					},
				},
			})
		}
		return detOutputs, nil
	case commonPB.Task_TASK_KEYPOINT:
		keypointResponse := postprocessResponse.(ray.KeypointOutput)
		var keypointOutputs []*modelPB.TaskOutput
		for i := range keypointResponse.Keypoints { // batch size
			var keypointObjects []*modelPB.KeypointObject
			for j := range keypointResponse.Keypoints[i] { // n keypoints in one image
				if keypointResponse.Scores[i][j] == -1 { // dummy object for batching to make sure every images have same output shape
					continue
				}
				var keypoints []*modelPB.Keypoint
				points := keypointResponse.Keypoints[i][j]
				for k := range points { // 17 point for each keypoint
					if points[k][0] == -1 && points[k][1] == -1 && points[k][2] == -1 { // dummy output for batching to make sure every images have same output shape
						continue
					}
					keypoints = append(keypoints, &modelPB.Keypoint{
						X: points[k][0],
						Y: points[k][1],
						V: points[k][2],
					})
				}
				keypointObjects = append(keypointObjects, &modelPB.KeypointObject{
					Keypoints: keypoints,
					BoundingBox: &modelPB.BoundingBox{
						Left:   keypointResponse.Boxes[i][j][0],
						Top:    keypointResponse.Boxes[i][j][1],
						Width:  keypointResponse.Boxes[i][j][2],
						Height: keypointResponse.Boxes[i][j][3],
					},
					Score: keypointResponse.Scores[i][j],
				})
			}
			keypointOutputs = append(keypointOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Keypoint{
					Keypoint: &modelPB.KeypointOutput{
						Objects: keypointObjects,
					},
				},
			})
		}
		if len(keypointOutputs) == 0 {
			keypointOutputs = append(keypointOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Keypoint{
					Keypoint: &modelPB.KeypointOutput{
						Objects: []*modelPB.KeypointObject{},
					},
				},
			})
		}
		return keypointOutputs, nil
	case commonPB.Task_TASK_OCR:
		ocrResponses := postprocessResponse.(ray.OcrOutput)
		batchedOutputDataBboxes := ocrResponses.Boxes
		batchedOutputDataTexts := ocrResponses.Texts
		batchedOutputDataScores := ocrResponses.Scores
		var ocrOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var ocrOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Ocr{
					Ocr: &modelPB.OcrOutput{
						Objects: []*modelPB.OcrObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				text := batchedOutputDataTexts[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if text != "" && box[0] != -1 {
					ocrOutput.GetOcr().Objects = append(ocrOutput.GetOcr().Objects, &modelPB.OcrObject{
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score: score,
						Text:  text,
					})
				}
			}
			ocrOutputs = append(ocrOutputs, &ocrOutput)
		}
		if len(ocrOutputs) == 0 {
			ocrOutputs = append(ocrOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Ocr{
					Ocr: &modelPB.OcrOutput{
						Objects: []*modelPB.OcrObject{},
					},
				},
			})
		}
		return ocrOutputs, nil

	case commonPB.Task_TASK_INSTANCE_SEGMENTATION:
		instanceSegmentationResponses := postprocessResponse.(ray.InstanceSegmentationOutput)
		batchedOutputDataRles := instanceSegmentationResponses.Rles
		batchedOutputDataBboxes := instanceSegmentationResponses.Boxes
		batchedOutputDataLabels := instanceSegmentationResponses.Labels
		batchedOutputDataScores := instanceSegmentationResponses.Scores
		var instanceSegmentationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var instanceSegmentationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelPB.InstanceSegmentationOutput{
						Objects: []*modelPB.InstanceSegmentationObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				rle := batchedOutputDataRles[i][j]
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if label != "" && rle != "" {
					instanceSegmentationOutput.GetInstanceSegmentation().Objects = append(instanceSegmentationOutput.GetInstanceSegmentation().Objects, &modelPB.InstanceSegmentationObject{
						Rle: rle,
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score:    score,
						Category: label,
					})
				}
			}
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &instanceSegmentationOutput)
		}
		if len(instanceSegmentationOutputs) == 0 {
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelPB.InstanceSegmentationOutput{
						Objects: []*modelPB.InstanceSegmentationObject{},
					},
				},
			})
		}
		return instanceSegmentationOutputs, nil

	case commonPB.Task_TASK_SEMANTIC_SEGMENTATION:
		semanticSegmentationResponses := postprocessResponse.(ray.SemanticSegmentationOutput)
		batchedOutputDataRles := semanticSegmentationResponses.Rles
		batchedOutputDataCategories := semanticSegmentationResponses.Categories
		var semanticSegmentationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataCategories { // loop over images
			var semanticSegmentationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelPB.SemanticSegmentationOutput{
						Stuffs: []*modelPB.SemanticSegmentationStuff{},
					},
				},
			}
			for j := range batchedOutputDataCategories[i] { // single image
				rle := batchedOutputDataRles[i][j]
				category := batchedOutputDataCategories[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if category != "" && rle != "" {
					semanticSegmentationOutput.GetSemanticSegmentation().Stuffs = append(semanticSegmentationOutput.GetSemanticSegmentation().Stuffs, &modelPB.SemanticSegmentationStuff{
						Rle:      rle,
						Category: category,
					})
				}
			}
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &semanticSegmentationOutput)
		}
		if len(semanticSegmentationOutputs) == 0 {
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelPB.SemanticSegmentationOutput{
						Stuffs: []*modelPB.SemanticSegmentationStuff{},
					},
				},
			})
		}
		return semanticSegmentationOutputs, nil
	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		textToImageResponses := postprocessResponse.(ray.TextToImageOutput)
		batchedOutputDataImages := textToImageResponses.Images
		var textToImageOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataImages { // loop over images
			var textToImageOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextToImage{
					TextToImage: &modelPB.TextToImageOutput{
						Images: batchedOutputDataImages[i],
					},
				},
			}

			textToImageOutputs = append(textToImageOutputs, &textToImageOutput)
		}
		if len(textToImageOutputs) == 0 {
			textToImageOutputs = append(textToImageOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextToImage{
					TextToImage: &modelPB.TextToImageOutput{
						Images: []string{},
					},
				},
			})
		}
		return textToImageOutputs, nil
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		imageToImageResponses := postprocessResponse.(ray.ImageToImageOutput)
		batchedOutputDataImages := imageToImageResponses.Images
		var imageToImageOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataImages { // loop over images
			var imageToImageOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_ImageToImage{
					ImageToImage: &modelPB.ImageToImageOutput{
						Images: batchedOutputDataImages[i],
					},
				},
			}

			imageToImageOutputs = append(imageToImageOutputs, &imageToImageOutput)
		}
		if len(imageToImageOutputs) == 0 {
			imageToImageOutputs = append(imageToImageOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_ImageToImage{
					ImageToImage: &modelPB.ImageToImageOutput{
						Images: []string{},
					},
				},
			})
		}
		return imageToImageOutputs, nil
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGenerationResponses := postprocessResponse.(ray.TextGenerationOutput)
		batchedOutputDataTexts := textGenerationResponses.Text
		var textGenerationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataTexts {
			var textGenerationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGeneration{
					TextGeneration: &modelPB.TextGenerationOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			textGenerationOutputs = append(textGenerationOutputs, &textGenerationOutput)
		}
		if len(textGenerationOutputs) == 0 {
			textGenerationOutputs = append(textGenerationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGeneration{
					TextGeneration: &modelPB.TextGenerationOutput{
						Text: "",
					},
				},
			})
		}
		return textGenerationOutputs, nil
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQuestionAnsweringResponses := postprocessResponse.(ray.VisualQuestionAnsweringOutput)
		batchedOutputDataTexts := visualQuestionAnsweringResponses.Text
		var visualQuestionAnsweringOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataTexts {
			var visualQuestionAnsweringOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_VisualQuestionAnswering{
					VisualQuestionAnswering: &modelPB.VisualQuestionAnsweringOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			visualQuestionAnsweringOutputs = append(visualQuestionAnsweringOutputs, &visualQuestionAnsweringOutput)
		}
		if len(visualQuestionAnsweringOutputs) == 0 {
			visualQuestionAnsweringOutputs = append(visualQuestionAnsweringOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_VisualQuestionAnswering{
					VisualQuestionAnswering: &modelPB.VisualQuestionAnsweringOutput{
						Text: "",
					},
				},
			})
		}
		return visualQuestionAnsweringOutputs, nil
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChatResponses := postprocessResponse.(ray.TextGenerationChatOutput)
		batchedOutputDataTexts := textGenerationChatResponses.Text
		var textGenerationChatOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataTexts {
			var textGenerationChatOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGenerationChat{
					TextGenerationChat: &modelPB.TextGenerationChatOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			textGenerationChatOutputs = append(textGenerationChatOutputs, &textGenerationChatOutput)
		}
		if len(textGenerationChatOutputs) == 0 {
			textGenerationChatOutputs = append(textGenerationChatOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGenerationChat{
					TextGenerationChat: &modelPB.TextGenerationChatOutput{
						Text: "",
					},
				},
			})
		}
		return textGenerationChatOutputs, nil
	default:
		outputs := postprocessResponse.([]ray.BatchUnspecifiedTaskOutputs)
		var rawOutputs []*modelPB.TaskOutput
		if len(outputs) == 0 {
			return []*modelPB.TaskOutput{}, nil
		}
		deserializedOutputs := outputs[0].SerializedOutputs
		for i := range deserializedOutputs {
			var singleImageOutput []*structpb.Struct

			for _, output := range outputs {
				unspecifiedOutput := ray.SingleOutputUnspecifiedTaskOutput{
					Name:     output.Name,
					Shape:    output.Shape,
					DataType: output.DataType,
					Data:     output.SerializedOutputs[i],
				}

				var mapOutput map[string]any
				b, err := json.Marshal(unspecifiedOutput)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(b, &mapOutput); err != nil {
					return nil, err
				}
				utils.ConvertAllJSONKeySnakeCase(mapOutput)

				b, err = json.Marshal(mapOutput)
				if err != nil {
					return nil, err
				}
				var structData = &structpb.Struct{}
				err = protojson.Unmarshal(b, structData)

				if err != nil {
					return nil, err
				}
				singleImageOutput = append(singleImageOutput, structData)
			}

			rawOutputs = append(rawOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Unspecified{
					Unspecified: &modelPB.UnspecifiedOutput{
						RawOutputs: singleImageOutput,
					},
				},
			})
		}
		if len(rawOutputs) == 0 {
			rawOutputs = append(rawOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Unspecified{
					Unspecified: &modelPB.UnspecifiedOutput{
						RawOutputs: []*structpb.Struct{},
					},
				},
			})
		}
		return rawOutputs, nil
	}
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
		if err = s.UpdateModelInstanceAdmin(ctx, ns, dbModel.ID, dbModel.Hardware, version.Version, false); err != nil {
			return err
		}
		if err = s.DeleteModelVersionAdmin(ctx, version); err != nil {
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
	if err := s.ray.UpdateContainerizedModel(ctx, name, ns.NsID, modelID, version, hardware != "cpu", isDeploy); err != nil {
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
