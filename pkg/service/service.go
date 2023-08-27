package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/model/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

// InferInput is the interface for the input to the model
type InferInput interface{}

// Service is the interface for the service layer
type Service interface {

	// Utils
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient
	GetRepository() repository.Repository
	GetRedisClient() *redis.Client
	GetUserUid(ctx context.Context) (uuid.UUID, error)
	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	PBModelToDBModel(ctx context.Context, owner string, pbModel *modelPB.Model) *datamodel.Model
	DBModelToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model) (*modelPB.Model, error)
	DBModelDefinitionToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) *modelPB.ModelDefinition

	ListModels(ctx context.Context, userUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error)
	ListUserModels(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error)
	GetUserModelByID(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string, view modelPB.View) (*datamodel.Model, error)
	GetUserModelByUID(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelUID uuid.UUID, view modelPB.View) (*datamodel.Model, error)
	DeleteUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) error
	RenameUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string, newModelID string) (*datamodel.Model, error)
	PublishUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) (*datamodel.Model, error)
	UnpublishUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) (*datamodel.Model, error)
	UpdateUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, model *datamodel.Model) (*datamodel.Model, error)
	UpdateUserModelState(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, model *datamodel.Model, state datamodel.ModelState) (*datamodel.Model, error)

	CreateUserModelAsync(ctx context.Context, model *datamodel.Model) (string, error)
	TriggerUserModel(ctx context.Context, modelUID uuid.UUID, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error)
	TriggerUserModelTestMode(ctx context.Context, modelUID uuid.UUID, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error)

	DeployUserModelAsync(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelUID uuid.UUID) (string, error)
	UndeployUserModelAsync(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelUID uuid.UUID) (string, error)

	GetModelDefinition(ctx context.Context, id string) (datamodel.ModelDefinition, error)
	GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (datamodel.ModelDefinition, error)
	ListModelDefinitions(ctx context.Context, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error)

	GetTritonEnsembleModel(ctx context.Context, modelUID uuid.UUID) (datamodel.TritonModel, error)
	GetTritonModels(ctx context.Context, modelUID uuid.UUID) ([]datamodel.TritonModel, error)

	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	// Private
	GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*datamodel.Model, error)
	GetModelByUIDAdmin(ctx context.Context, modelUID uuid.UUID, view modelPB.View) (*datamodel.Model, error)
	ListModelsAdmin(ctx context.Context, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error)
	DeployUserModelAsyncAdmin(ctx context.Context, modelUID uuid.UUID) (string, error)
	UndeployUserModelAsyncAdmin(ctx context.Context, userUID uuid.UUID, modelUID uuid.UUID) (string, error)
	CheckModel(ctx context.Context, modelUID uuid.UUID) (*modelPB.Model_State, error)

	// Controller
	GetResourceState(ctx context.Context, modelUID uuid.UUID) (*modelPB.Model_State, error)
	UpdateResourceState(ctx context.Context, modelUID uuid.UUID, state modelPB.Model_State, progress *int32, workflowID *string) error
	DeleteResourceState(ctx context.Context, modelUID uuid.UUID) error

	// Usage collection
	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) error
}

type service struct {
	repository               repository.Repository
	triton                   triton.Triton
	redisClient              *redis.Client
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	temporalClient           client.Client
	controllerClient         controllerPB.ControllerPrivateServiceClient
	defaultUserUid           uuid.UUID
}

// NewService returns a new service instance
func NewService(r repository.Repository, t triton.Triton, m mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, tc client.Client, cs controllerPB.ControllerPrivateServiceClient, defaultUserUid uuid.UUID) Service {
	return &service{
		repository:               r,
		triton:                   t,
		mgmtPrivateServiceClient: m,
		redisClient:              rc,
		temporalClient:           tc,
		controllerClient:         cs,
		defaultUserUid:           defaultUserUid,
	}
}

func (s *service) GetRepository() repository.Repository {
	return s.repository
}

// GetRedisClient returns the redis client
func (s *service) GetRedisClient() *redis.Client {
	return s.redisClient
}

// GetMgmtPrivateServiceClient returns the management private service client
func (s *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return s.mgmtPrivateServiceClient
}

// GetUserPermalink returns the api user
func (s *service) GetUserUid(ctx context.Context) (uuid.UUID, error) {

	// Verify if "authorization" is in the header
	authorization := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthorization)
	// TODO: temporary solution to restrict cloud version from calling APIs without header
	// need further concrete design of authentication
	if strings.HasPrefix(config.Config.Server.Edition, "cloud") && authorization == "" {
		return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}
	apiToken := strings.Replace(authorization, "Bearer ", "", 1)
	if apiToken != "" {
		ownerPermalink, err := s.redisClient.Get(context.Background(), fmt.Sprintf(constant.AccessTokenKeyFormat, apiToken)).Result()
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: ownerPermalink})
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		return uuid.FromStringOrNil(*resp.User.Uid), nil
	}
	// Verify if "jwt-sub" is in the header
	headerUserUId := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if headerUserUId != "" {
		_, err := uuid.FromString(headerUserUId)
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}
		_, err = s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: "users/" + headerUserUId})
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		return uuid.FromStringOrNil(headerUserUId), nil
	}

	return s.defaultUserUid, nil
}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
	if err != nil {
		return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
	}
	return fmt.Sprintf("users/%s", userResp.User.Id), nil
}
func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtPB.GetUserAdminRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("ConvertOwnerNameToPermalink error")
	}
	return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
}

func (s *service) GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error) {

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))

	if err != nil {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink((fmt.Sprintf("%s/%s", splits[0], splits[1])))
	if err != nil {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

// TODO: determine the necessity of this block of codes
// func (s *service) DeployModel(modelUID uuid.UUID) error {
// 	var tEnsembleModel datamodel.TritonModel
// 	var err error

// 	if tEnsembleModel, err = s.repository.GetTritonEnsembleModel(modelUID); err != nil {
// 		return err
// 	}
// 	// Load one ensemble model, which will also load all its dependent models
// 	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
// 		if err1 := s.repository.UpdateModel(modelUID, datamodel.Model{
// 			State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
// 		}); err1 != nil {
// 			return err1
// 		}
// 		return err
// 	}

// 	if err = s.repository.UpdateModel(modelUID, datamodel.Model{
// 		State: datamodel.ModelState(modelPB.Model_STATE_ONLINE),
// 	}); err != nil {
// 		return err
// 	}

// 	return nil
// }

func (s *service) UndeployModel(ctx context.Context, ownerPermalink string, userPermalink string, modelUID uuid.UUID) error {

	// var tritonModels []datamodel.TritonModel
	var err error

	if _, err = s.repository.GetTritonModels(modelUID); err != nil {
		return err
	}

	if err := s.repository.UpdateUserModel(ownerPermalink, userPermalink, modelUID, &datamodel.Model{
		State: datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) GetUserModelByID(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string, view modelPB.View) (*datamodel.Model, error) {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)
	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelID, view)
}

func (s *service) GetUserModelByUID(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelUID uuid.UUID, view modelPB.View) (*datamodel.Model, error) {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)
	return s.repository.GetUserModelByUID(ctx, ownerPermalink, userPermalink, modelUID, view)
}

func (s *service) GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*datamodel.Model, error) {
	return s.repository.GetModelByIDAdmin(ctx, modelID, view)
}

func (s *service) GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, view modelPB.View) (*datamodel.Model, error) {
	return s.repository.GetModelByUIDAdmin(ctx, uid, view)
}

func (s *service) TriggerUserModelTestMode(ctx context.Context, modelUID uuid.UUID, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error) {

	// switch task {
	// case commonPB.Task_TASK_CLASSIFICATION,
	// 	commonPB.Task_TASK_DETECTION,
	// 	commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	// 	commonPB.Task_TASK_KEYPOINT,
	// 	commonPB.Task_TASK_OCR,
	// 	commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	// 	commonPB.Task_TASK_UNSPECIFIED:

	// 	if strings.HasPrefix(owner, "users/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), int64(len(inferInput.([][]byte))))
	// 	} else if strings.HasPrefix(owner, "orgs/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), int64(len(inferInput.([][]byte))))
	// 	}
	// case commonPB.Task_TASK_TEXT_TO_IMAGE:
	// 	if strings.HasPrefix(owner, "users/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), 1)
	// 	} else if strings.HasPrefix(owner, "orgs/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), 1)
	// 	}
	// case commonPB.Task_TASK_TEXT_GENERATION:
	// 	if strings.HasPrefix(owner, "users/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), 1)
	// 	} else if strings.HasPrefix(owner, "orgs/") {
	// 		s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), 1)
	// 	}
	// default:
	// 	return nil, fmt.Errorf("unknown task input type")
	// }

	return s.TriggerUserModel(ctx, modelUID, inferInput, task)
}

func (s *service) CheckModel(ctx context.Context, modelUID uuid.UUID) (*modelPB.Model_State, error) {
	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelUID)
	if err != nil {
		return nil, fmt.Errorf("triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelReadyResponse := s.triton.ModelReadyRequest(ctx, ensembleModelName, fmt.Sprint(ensembleModelVersion))

	state := modelPB.Model_STATE_UNSPECIFIED
	if modelReadyResponse == nil {
		state = modelPB.Model_STATE_ERROR
	} else if modelReadyResponse.Ready {
		state = modelPB.Model_STATE_ONLINE
	} else {
		state = modelPB.Model_STATE_OFFLINE
	}

	return &state, nil
}

func (s *service) TriggerUserModel(ctx context.Context, modelUID uuid.UUID, inferInput InferInput, task commonPB.Task) ([]*modelPB.TaskOutput, error) {

	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelUID)
	if err != nil {
		return nil, fmt.Errorf("triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelMetadataResponse := s.triton.ModelMetadataRequest(ctx, ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, fmt.Errorf("model is offline")
	}
	modelConfigResponse := s.triton.ModelConfigRequest(ctx, ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, err
	}

	// We use a simple model that takes 2 input tensors of 16 integers
	// each and returns 2 output tensors of 16 integers each. One
	// output tensor is the element-wise sum of the inputs and one
	// output is the element-wise difference.
	inferResponse, err := s.triton.ModelInferRequest(ctx, task, inferInput, ensembleModelName, fmt.Sprint(ensembleModelVersion), modelMetadataResponse, modelConfigResponse)
	if err != nil {
		return nil, err
	}

	// We expect there to be 2 results (each with batch-size 1). Walk
	// over all 16 result elements and print the sum and difference
	// calculated by the modelPB.
	postprocessResponse, err := s.triton.PostProcess(inferResponse, modelMetadataResponse, task)
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
		detResponses := postprocessResponse.(triton.DetectionOutput)
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
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Triton to be able to batch Tensors
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
		keypointResponse := postprocessResponse.(triton.KeypointOutput)
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
		ocrResponses := postprocessResponse.(triton.OcrOutput)
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
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
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
		instanceSegmentationResponses := postprocessResponse.(triton.InstanceSegmentationOutput)
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
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
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
		semanticSegmentationResponses := postprocessResponse.(triton.SemanticSegmentationOutput)
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
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
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
		textToImageResponses := postprocessResponse.(triton.TextToImageOutput)
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
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGenerationResponses := postprocessResponse.(triton.TextGenerationOutput)
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
	default:
		outputs := postprocessResponse.([]triton.BatchUnspecifiedTaskOutputs)
		var rawOutputs []*modelPB.TaskOutput
		if len(outputs) == 0 {
			return []*modelPB.TaskOutput{}, nil
		}
		deserializedOutputs := outputs[0].SerializedOutputs
		for i := range deserializedOutputs {
			var singleImageOutput []*structpb.Struct

			for _, output := range outputs {
				unspecifiedOutput := triton.SingleOutputUnspecifiedTaskOutput{
					Name:     output.Name,
					Shape:    output.Shape,
					DataType: output.DataType,
					Data:     output.SerializedOutputs[i],
				}

				var mapOutput map[string]interface{}
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

func (s *service) ListModels(ctx context.Context, userUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error) {
	userPermalLink := resource.UserUidToUserPermalink(userUID)
	return s.repository.ListModels(ctx, userPermalLink, view, pageSize, pageToken)
}

func (s *service) ListUserModels(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error) {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)
	return s.repository.ListUserModels(ctx, ownerPermalink, userPermalink, view, pageSize, pageToken)
}

func (s *service) ListModelsAdmin(ctx context.Context, view modelPB.View, pageSize int, pageToken string) ([]*datamodel.Model, string, int64, error) {
	return s.repository.ListModelsAdmin(ctx, view, pageSize, pageToken)
}

func (s *service) DeleteUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) error {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	state, err := s.GetResourceState(ctx, modelInDB.UID)
	if err != nil {
		return err
	}

	if *state == modelPB.Model_STATE_UNSPECIFIED {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s", modelInDB.ID),
					Description: "The model is still in operations, please wait the operation finish and try it again",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if err := s.UndeployModel(ctx, ownerPermalink, userPermalink, modelInDB.UID); err != nil {
		return err
	}

	// remove README.md
	_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md", config.Config.TritonServer.ModelStore, ownerPermalink, modelInDB.ID))
	tritonModels, err := s.repository.GetTritonModels(modelInDB.UID)
	if err == nil {
		// remove model folders
		for i := 0; i < len(tritonModels); i++ {
			modelDir := filepath.Join(config.Config.TritonServer.ModelStore, tritonModels[i].Name)
			_ = os.RemoveAll(modelDir)
		}
	}

	for state.String() == modelPB.Model_STATE_ONLINE.String() {
		if state, err = s.GetResourceState(ctx, modelInDB.UID); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := s.DeleteResourceState(ctx, modelInDB.UID); err != nil {
		return err
	}

	return s.repository.DeleteModel(modelInDB.UID)
}

func (s *service) RenameUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string, newModelID string) (*datamodel.Model, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	err = s.repository.UpdateUserModel(ownerPermalink, userPermalink, modelInDB.UID, &datamodel.Model{
		ID: newModelID,
	})
	if err != nil {
		return nil, err
	}

	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, newModelID, modelPB.View_VIEW_FULL)
}

func (s *service) PublishUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) (*datamodel.Model, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	err = s.repository.UpdateUserModel(ownerPermalink, userPermalink, modelInDB.UID, &datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC),
	})
	if err != nil {
		return nil, err
	}

	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelInDB.ID, modelPB.View_VIEW_FULL)
}

func (s *service) UnpublishUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, modelID string) (*datamodel.Model, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	err = s.repository.UpdateUserModel(ownerPermalink, userPermalink, modelInDB.UID, &datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE),
	})
	if err != nil {
		return nil, err
	}

	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelInDB.ID, modelPB.View_VIEW_FULL)
}

func (s *service) UpdateUserModel(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, model *datamodel.Model) (*datamodel.Model, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, model.ID, modelPB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	err = s.repository.UpdateUserModel(ownerPermalink, userPermalink, modelInDB.UID, model)
	if err != nil {
		return nil, err
	}

	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelInDB.ID, modelPB.View_VIEW_FULL)
}

// TODO: gorm do not update the zero value with struct, so we need to update the state manually.
func (s *service) UpdateUserModelState(ctx context.Context, ns resource.Namespace, userUID uuid.UUID, model *datamodel.Model, state datamodel.ModelState) (*datamodel.Model, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUID)

	modelInDB, err := s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, model.ID, modelPB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UpdateUserModelState(ownerPermalink, userPermalink, modelInDB.UID, state); err != nil {
		return nil, err
	}

	return s.repository.GetUserModelByID(ctx, ownerPermalink, userPermalink, modelInDB.ID, modelPB.View_VIEW_FULL)
}

func (s *service) GetModelDefinition(ctx context.Context, id string) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinition(id)
}

func (s *service) GetModelDefinitionByUID(ctx context.Context, uid uuid.UUID) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinitionByUID(uid)
}

func (s *service) ListModelDefinitions(ctx context.Context, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error) {
	return s.repository.ListModelDefinitions(view, pageSize, pageToken)
}

func (s *service) GetTritonEnsembleModel(ctx context.Context, modelUID uuid.UUID) (datamodel.TritonModel, error) {
	return s.repository.GetTritonEnsembleModel(modelUID)
}

func (s *service) GetTritonModels(ctx context.Context, modelUID uuid.UUID) ([]datamodel.TritonModel, error) {
	return s.repository.GetTritonModels(modelUID)
}
