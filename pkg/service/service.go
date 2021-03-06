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

	"github.com/go-redis/redis/v9"

	"github.com/gofrs/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/x/sterr"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type Service interface {
	CreateModel(owner string, model *datamodel.Model) (*datamodel.Model, error)
	GetModelById(owner string, modelID string, view modelPB.View) (datamodel.Model, error)
	GetModelByUid(owner string, modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error)
	DeleteModel(owner string, modelID string) error
	RenameModel(owner string, modelID string, newModelId string) (datamodel.Model, error)
	PublishModel(owner string, modelID string) (datamodel.Model, error)
	UnpublishModel(owner string, modelID string) (datamodel.Model, error)
	UpdateModel(modelUID uuid.UUID, model *datamodel.Model) (datamodel.Model, error)
	ListModel(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error)
	ModelInfer(modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error)
	ModelInferTestMode(owner string, modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error)
	GetModelInstance(modelUID uuid.UUID, instanceID string, view modelPB.View) (datamodel.ModelInstance, error)
	GetModelInstanceByUid(modelUID uuid.UUID, instanceUID uuid.UUID, view modelPB.View) (datamodel.ModelInstance, error)
	ListModelInstance(modelUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelInstance, string, int64, error)
	DeployModelInstance(modelInstanceID uuid.UUID) error
	UndeployModelInstance(modelInstanceID uuid.UUID) error
	GetModelDefinition(id string) (datamodel.ModelDefinition, error)
	GetModelDefinitionByUid(uid uuid.UUID) (datamodel.ModelDefinition, error)
	ListModelDefinition(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error)
	GetTritonEnsembleModel(modelInstanceUID uuid.UUID) (datamodel.TritonModel, error)
	GetTritonModels(modelInstanceUID uuid.UUID) ([]datamodel.TritonModel, error)
}

type service struct {
	repository            repository.Repository
	triton                triton.Triton
	redisClient           *redis.Client
	pipelineServiceClient pipelinePB.PipelineServiceClient
}

func NewService(r repository.Repository, t triton.Triton, p pipelinePB.PipelineServiceClient, rc *redis.Client) Service {
	return &service{
		repository:            r,
		triton:                t,
		pipelineServiceClient: p,
		redisClient:           rc,
	}
}

func (s *service) DeployModelInstance(modelInstanceID uuid.UUID) error {
	var tEnsembleModel datamodel.TritonModel
	var err error

	if tEnsembleModel, err = s.repository.GetTritonEnsembleModel(modelInstanceID); err != nil {
		return err
	}
	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err1 := s.repository.UpdateModelInstance(modelInstanceID, datamodel.ModelInstance{
			State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ERROR),
		}); err1 != nil {
			return err1
		}
		return err
	}

	if err = s.repository.UpdateModelInstance(modelInstanceID, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ONLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) UndeployModelInstance(modelInstanceID uuid.UUID) error {

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = s.repository.GetTritonModels(modelInstanceID); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err1 := s.repository.UpdateModelInstance(modelInstanceID, datamodel.ModelInstance{
				State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ERROR),
			}); err1 != nil {
				return err1
			}
			return err
		}
	}

	if err := s.repository.UpdateModelInstance(modelInstanceID, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) GetModelById(owner string, modelID string, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelById(owner, modelID, view)
}

func (s *service) GetModelByUid(owner string, uid uuid.UUID, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelByUid(owner, uid, view)
}

func (s *service) ModelInferTestMode(owner string, modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error) {
	// Increment trigger image numbers
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	uid, _ := resource.GetPermalinkUID(owner)
	if strings.HasPrefix(owner, "users/") {
		s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.image.num", uid), int64(len(imgsBytes)))
	} else if strings.HasPrefix(owner, "orgs/") {
		s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.image.num", uid), int64(len(imgsBytes)))
	}
	return s.ModelInfer(modelInstanceUID, imgsBytes, task)
}

func (s *service) ModelInfer(modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error) {

	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelInstanceUID)
	if err != nil {
		return nil, fmt.Errorf("triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelMetadataResponse := s.triton.ModelMetadataRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, fmt.Errorf("model is offline")
	}
	modelConfigResponse := s.triton.ModelConfigRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, err
	}

	// We use a simple model that takes 2 input tensors of 16 integers
	// each and returns 2 output tensors of 16 integers each. One
	// output tensor is the element-wise sum of the inputs and one
	// output is the element-wise difference.
	inferResponse, err := s.triton.ModelInferRequest(task, imgsBytes, ensembleModelName, fmt.Sprint(ensembleModelVersion), modelMetadataResponse, modelConfigResponse)
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
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		clsResponses := postprocessResponse.([]string)
		var contents []*modelPB.ClassificationOutput
		for _, clsRes := range clsResponses {
			clsResSplit := strings.Split(clsRes, ":")
			if len(clsResSplit) == 2 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.ClassificationOutput{
					Category: clsResSplit[1],
					Score:    float32(score),
				}
				contents = append(contents, &clsOutput)
			} else if len(clsResSplit) == 3 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.ClassificationOutput{
					Category: clsResSplit[2],
					Score:    float32(score),
				}
				contents = append(contents, &clsOutput)
			} else {
				return nil, fmt.Errorf("unable to decode inference output")
			}
		}
		clsOutputs := modelPB.ClassificationOutputs{
			ClassificationOutputs: contents,
		}
		return &clsOutputs, nil

	case modelPB.ModelInstance_TASK_DETECTION:
		detResponses := postprocessResponse.(triton.DetectionOutput)
		batchedOutputDataBboxes := detResponses.Boxes
		batchedOutputDataLabels := detResponses.Labels
		var detOutputs modelPB.DetectionOutputs
		for i := range batchedOutputDataBboxes {
			var contents []*modelPB.BoundingBoxObject
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Triton to be able to batch Tensors
				if label != "0" {
					pred := &modelPB.BoundingBoxObject{
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

					contents = append(contents, pred)
				}
			}
			detOutput := &modelPB.DetectionOutput{
				BoundingBoxObjects: contents,
			}
			detOutputs.DetectionOutputs = append(detOutputs.DetectionOutputs, detOutput)
		}
		return &detOutputs, nil
	case modelPB.ModelInstance_TASK_KEYPOINT:
		keypointResponse := postprocessResponse.(triton.KeypointOutput)
		var contents []*modelPB.KeypointObject
		for i := range keypointResponse.Keypoints {
			score := keypointResponse.Scores[i]
			var keypoints []*modelPB.Keypoint
			for j := range keypointResponse.Keypoints[i] {
				keypoint := keypointResponse.Keypoints[i][j]
				keypoints = append(keypoints, &modelPB.Keypoint{
					X: keypoint[0],
					Y: keypoint[1],
					V: keypoint[2],
				})
			}
			contents = append(contents, &modelPB.KeypointObject{
				Score:     score,
				Keypoints: keypoints,
			})
		}
		keypointOutputs := modelPB.KeypointOutputs{
			KeypointOutputs: contents,
		}
		return &keypointOutputs, nil
	default:
		outputs := postprocessResponse.([]triton.BatchUnspecifiedTaskOutputs)
		var rawOutputs []*structpb.Struct
		if len(outputs) == 0 {
			return &modelPB.UnspecifiedTaskOutputs{
				RawOutputs: rawOutputs,
			}, nil
		}

		deserializedOutputs := outputs[0].SerializedOutputs
		for i := range deserializedOutputs {
			var singleImageOutput []triton.SingleOutputUnspecifiedTaskOutput
			for _, output := range outputs {
				singleImageOutput = append(singleImageOutput, triton.SingleOutputUnspecifiedTaskOutput{
					Name:     output.Name,
					Shape:    output.Shape,
					DataType: output.DataType,
					Data:     output.SerializedOutputs[i],
				})
			}
			output := triton.UnspecifiedTaskOutput{
				RawOutput: singleImageOutput,
			}
			var mapOutput map[string][]map[string]interface{}
			b, err := json.Marshal(output)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &mapOutput); err != nil {
				return nil, err
			}
			util.ConvertAllJSONKeySnakeCase(mapOutput)

			b, err = json.Marshal(mapOutput)
			if err != nil {
				return nil, err
			}
			var data = &structpb.Struct{}
			err = protojson.Unmarshal(b, data)

			if err != nil {
				return nil, err
			}
			rawOutputs = append(rawOutputs, data)
		}
		return &modelPB.UnspecifiedTaskOutputs{
			RawOutputs: rawOutputs,
		}, nil
	}
}

func (s *service) CreateModel(owner string, model *datamodel.Model) (*datamodel.Model, error) {
	if existingModel, _ := s.repository.GetModelById(model.Owner, model.ID, modelPB.View_VIEW_FULL); existingModel.ID != "" {
		return &datamodel.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your workspace", model.ID)
	}
	if err := s.repository.CreateModel(*model); err != nil {
		return &datamodel.Model{}, err
	}

	if createdModel, err := s.repository.GetModelById(model.Owner, model.ID, modelPB.View_VIEW_FULL); err != nil {
		return &datamodel.Model{}, err
	} else {
		return &createdModel, nil
	}
}

func (s *service) ListModel(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error) {
	return s.repository.ListModel(owner, view, pageSize, pageToken)
}

func (s *service) DeleteModel(owner string, modelID string) error {
	logger, _ := logger.GetZapLogger()

	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	filter := fmt.Sprintf("recipe.model_instances:\"models/%s\"", modelInDB.UID)

	pipeResp, err := s.pipelineServiceClient.ListPipeline(context.Background(), &pipelinePB.ListPipelineRequest{
		Filter: &filter,
	})
	if err != nil {
		return err
	}
	if len(pipeResp.Pipelines) > 0 {
		var pipeIDs []string
		for _, pipe := range pipeResp.Pipelines {
			pipeIDs = append(pipeIDs, pipe.GetId())
		}
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s", modelInDB.ID),
					Description: fmt.Sprintf("The model is still in use by pipeline: %s ", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	modelInstancesInDB, err := s.repository.GetModelInstances(modelInDB.UID)
	if err == nil {
		for i := 0; i < len(modelInstancesInDB); i++ {
			if err := s.UndeployModelInstance(modelInstancesInDB[i].UID); err != nil {
				return err
			}
			// remove README.md
			_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md#%v", config.Config.TritonServer.ModelStore, owner, modelInDB.ID, modelInstancesInDB[i].ID))
			tritonModels, err := s.repository.GetTritonModels(modelInstancesInDB[i].UID)
			if err == nil {
				// remove model folders
				for i := 0; i < len(tritonModels); i++ {
					modelDir := filepath.Join(config.Config.TritonServer.ModelStore, tritonModels[i].Name)
					_ = os.RemoveAll(modelDir)
				}
			}
		}
	}

	return s.repository.DeleteModel(modelInDB.UID)
}

func (s *service) RenameModel(owner string, modelID string, newModelId string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID: newModelId,
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, newModelId, modelPB.View_VIEW_FULL)
}

func (s *service) PublishModel(owner string, modelID string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC),
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
}

func (s *service) UnpublishModel(owner string, modelID string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE),
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
}

func (s *service) UpdateModel(modelUID uuid.UUID, model *datamodel.Model) (datamodel.Model, error) {
	err := s.repository.UpdateModel(modelUID, *model)
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(model.Owner, model.ID, modelPB.View_VIEW_FULL)
}

func (s *service) GetModelInstance(modelUID uuid.UUID, modelInstanceID string, view modelPB.View) (datamodel.ModelInstance, error) {
	return s.repository.GetModelInstance(modelUID, modelInstanceID, view)
}

func (s *service) GetModelInstanceByUid(modelUID uuid.UUID, modelInstanceUid uuid.UUID, view modelPB.View) (datamodel.ModelInstance, error) {
	return s.repository.GetModelInstanceByUid(modelUID, modelInstanceUid, view)
}

func (s *service) ListModelInstance(modelUID uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelInstance, string, int64, error) {
	return s.repository.ListModelInstance(modelUID, view, pageSize, pageToken)
}

func (s *service) GetModelDefinition(id string) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinition(id)
}

func (s *service) GetModelDefinitionByUid(uid uuid.UUID) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinitionByUid(uid)
}

func (s *service) ListModelDefinition(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error) {
	return s.repository.ListModelDefinition(view, pageSize, pageToken)
}

func (s *service) GetTritonEnsembleModel(modelInstanceUID uuid.UUID) (datamodel.TritonModel, error) {
	return s.repository.GetTritonEnsembleModel(modelInstanceUID)
}

func (s *service) GetTritonModels(modelInstanceUID uuid.UUID) ([]datamodel.TritonModel, error) {
	return s.repository.GetTritonModels(modelInstanceUID)
}
