package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

type Service interface {
	CreateModel(owner string, model *datamodel.Model) (*datamodel.Model, error)
	GetModelById(owner string, modelId string) (datamodel.Model, error)
	DeleteModel(owner string, modelId string) error
	RenameModel(owner string, modelId string, newModelId string) (datamodel.Model, error)
	ListModel(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error)
	ModelInfer(modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error)
	GetModelInstance(modelUid uuid.UUID, instanceId string) (datamodel.ModelInstance, error)
	ListModelInstance(modelUid uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelInstance, string, int64, error)
	DeployModelInstance(modelInstanceId uuid.UUID) error
	UndeployModelInstance(modelInstanceId uuid.UUID) error
	GetModelDefinition(id string) (datamodel.ModelDefinition, error)
	ListModelDefinition(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error)
}

type service struct {
	repository repository.Repository
	triton     triton.Triton
}

func NewService(r repository.Repository, t triton.Triton) Service {
	return &service{
		repository: r,
		triton:     t,
	}
}

func (s *service) DeployModelInstance(modelInstanceId uuid.UUID) error {
	var tEnsembleModel datamodel.TritonModel
	var err error

	if tEnsembleModel, err = s.repository.GetTritonEnsembleModel(modelInstanceId); err != nil {
		return err
	}
	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err = s.repository.UpdateModelInstance(modelInstanceId, datamodel.ModelInstance{
			State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ERROR),
		}); err != nil {
			return err
		}
		return err
	}

	if err = s.repository.UpdateModelInstance(modelInstanceId, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ONLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) UndeployModelInstance(modelInstanceId uuid.UUID) error {

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = s.repository.GetTritonModels(modelInstanceId); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err = s.repository.UpdateModelInstance(modelInstanceId, datamodel.ModelInstance{
				State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ERROR),
			}); err != nil {
				return err
			}
			return err
		}
	}

	if err := s.repository.UpdateModelInstance(modelInstanceId, datamodel.ModelInstance{
		State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) GetModelById(owner string, modelId string) (datamodel.Model, error) {
	return s.repository.GetModelById(owner, modelId)
}

func (s *service) ModelInfer(modelInstanceUID uuid.UUID, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error) {
	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelInstanceUID)
	if err != nil {
		return nil, fmt.Errorf("Triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelMetadataResponse := s.triton.ModelMetadataRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, fmt.Errorf("Model is offline")
	}
	modelConfigResponse := s.triton.ModelConfigRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, err
	}
	// /* We use a simple model that takes 2 input tensors of 16 integers
	// each and returns 2 output tensors of 16 integers each. One
	// output tensor is the element-wise sum of the inputs and one
	// output is the element-wise difference. */
	inferResponse, err := s.triton.ModelInferRequest(task, imgsBytes, ensembleModelName, fmt.Sprint(ensembleModelVersion), modelMetadataResponse, modelConfigResponse)
	if err != nil {
		return nil, err
	}
	// /* We expect there to be 2 results (each with batch-size 1). Walk
	// over all 16 result elements and print the sum and difference
	// calculated by the modelPB. */
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
			if len(clsResSplit) != 3 {
				return nil, fmt.Errorf("Unable to decode inference output")
			}
			score, err := strconv.ParseFloat(clsResSplit[0], 32)
			if err != nil {
				return nil, fmt.Errorf("Unable to decode inference output")
			}
			clsOutput := modelPB.ClassificationOutput{
				Category: clsResSplit[2],
				Score:    float32(score),
			}
			contents = append(contents, &clsOutput)
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
	default:
		return postprocessResponse, nil
	}
}

func (s *service) CreateModel(owner string, model *datamodel.Model) (*datamodel.Model, error) {
	if existingModel, _ := s.repository.GetModelById(model.Owner, model.ID); existingModel.ID != "" {
		return &datamodel.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your workspace", model.ID)
	}
	if err := s.repository.CreateModel(*model); err != nil {
		return &datamodel.Model{}, err
	}

	if createdModel, err := s.repository.GetModelById(model.Owner, model.ID); err != nil {
		return &datamodel.Model{}, err
	} else {
		return &createdModel, nil
	}
}

func (s *service) ListModel(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error) {
	return s.repository.ListModel(owner, view, pageSize, pageToken)
}

func (s *service) DeleteModel(owner string, modelId string) error {
	modelInDB, err := s.GetModelById(owner, modelId)
	if err != nil {
		return err
	}
	modelInstancesInDB, err := s.repository.GetModelInstances(modelInDB.UID)
	if err == nil {
		for i := 0; i < len(modelInstancesInDB); i++ {
			if err := s.UndeployModelInstance(modelInstancesInDB[i].UID); err != nil {
				return err
			}
			// remove README.md
			_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md#%v", configs.Config.TritonServer.ModelStore, owner, modelInDB.ID, modelInstancesInDB[i].ID))
			tritonModels, err := s.repository.GetTritonModels(modelInstancesInDB[i].UID)
			if err == nil {
				// remove model folders
				for i := 0; i < len(tritonModels); i++ {
					modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
					_ = os.RemoveAll(modelDir)
				}
			}
		}
	}

	return s.repository.DeleteModel(modelInDB.UID)
}

func (s *service) RenameModel(owner string, modelId string, newModelId string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelId)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID: newModelId,
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, newModelId)
}

func (s *service) GetModelInstance(modelUid uuid.UUID, modelInstanceId string) (datamodel.ModelInstance, error) {
	return s.repository.GetModelInstance(modelUid, modelInstanceId)
}

func (s *service) ListModelInstance(modelUid uuid.UUID, view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelInstance, string, int64, error) {
	return s.repository.ListModelInstance(modelUid, view, pageSize, pageToken)
}

func (s *service) GetModelDefinition(id string) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinition(id)
}

func (s *service) ListModelDefinition(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error) {
	return s.repository.ListModelDefinition(view, pageSize, pageToken)
}
