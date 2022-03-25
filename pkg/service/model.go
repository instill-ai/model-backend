package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/triton"
	modelDB "github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelService interface {
	CreateModel(model *modelDB.Model) (*modelDB.Model, error)
	GetModelByName(namespace string, modelName string) (modelDB.Model, error)
	CreateVersion(version modelDB.Version) (modelDB.Version, error)
	GetModelVersion(modelId uint64, version uint64) (modelDB.Version, error)
	GetModelVersions(modelId uint64) ([]modelDB.Version, error)
	GetModelVersionLatest(modelId uint64) (modelDB.Version, error)
	GetFullModelData(namespace string, modelName string) (*modelPB.Model, error)
	ModelInfer(namespace string, modelName string, version uint64, imgsBytes [][]byte, task modelPB.Model_Task) (interface{}, error)
	CreateModelBinaryFileUpload(namespace string, createdModel *modelDB.Model) (*modelPB.Model, error)
	ListModels(namespace string) ([]*modelPB.Model, error)
	UpdateModelVersion(namespace string, updatedInfo *modelPB.UpdateModelVersionRequest) (*modelPB.ModelVersion, error)
	DeleteModel(namespace string, modelName string) error
	DeleteModelVersion(namespace string, modelName string, version uint64) error
}

type modelService struct {
	modelRepository repository.ModelRepository
	triton          triton.TritonService
}

func NewModelService(r repository.ModelRepository, t triton.TritonService) ModelService {
	return &modelService{
		modelRepository: r,
		triton:          t,
	}
}

func createModelVersion(modelVersionInDB modelDB.Version) *modelPB.ModelVersion {
	var st = modelPB.ModelVersion_STATUS_OFFLINE
	if modelVersionInDB.Status == modelPB.ModelVersion_STATUS_ONLINE.String() {
		st = modelPB.ModelVersion_STATUS_ONLINE
	} else if modelVersionInDB.Status == modelPB.ModelVersion_STATUS_ERROR.String() {
		st = modelPB.ModelVersion_STATUS_ERROR
	}

	return &modelPB.ModelVersion{
		Version:     modelVersionInDB.Version,
		ModelId:     modelVersionInDB.ModelId,
		Description: modelVersionInDB.Description,
		CreatedAt:   timestamppb.New(modelVersionInDB.CreatedAt),
		UpdatedAt:   timestamppb.New(modelVersionInDB.UpdatedAt),
		Status:      st,
	}
}

func createModelInfo(modelInDB modelDB.Model, versions []modelDB.Version, tritonModels []modelDB.TModel) *modelPB.Model {
	var vers []*modelPB.ModelVersion
	for i := 0; i < len(versions); i++ {
		vers = append(vers, createModelVersion(versions[i]))
	}
	return &modelPB.Model{
		Name:          modelInDB.Name,
		FullName:      modelInDB.FullName,
		Id:            uint64(modelInDB.Id),
		Task:          modelPB.Model_Task(modelInDB.Task),
		ModelVersions: vers,
	}
}

func setModelOnline(s *modelService, modelID uint64, modelVersion uint64) error {
	var tEnsembleModel modelDB.TModel
	var err error

	if tEnsembleModel, err = s.modelRepository.GetTEnsembleModel(modelID, modelVersion); err != nil {
		return err
	}

	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err = s.modelRepository.UpdateModelVersion(modelID, tEnsembleModel.ModelVersion, modelDB.Version{
			UpdatedAt: time.Now(),
			Status:    modelPB.ModelVersion_STATUS_ERROR.String(),
		}); err != nil {
			return err
		}
		return err
	}

	if err = s.modelRepository.UpdateModelVersion(modelID, tEnsembleModel.ModelVersion, modelDB.Version{
		UpdatedAt: time.Now(),
		Status:    modelPB.ModelVersion_STATUS_ONLINE.String(),
	}); err != nil {
		return err
	}

	return nil
}

func setModelOffline(s *modelService, modelID uint64, modelVersion uint64) error {

	var tritonModels []modelDB.TModel
	var err error

	if tritonModels, err = s.modelRepository.GetTModels(modelID); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err = s.modelRepository.UpdateModelVersion(modelID, modelVersion, modelDB.Version{
				UpdatedAt: time.Now(),
				Status:    modelPB.ModelVersion_STATUS_ERROR.String(),
			}); err != nil {
				return err
			}
			return err
		}
	}

	if err := s.modelRepository.UpdateModelVersion(modelID, modelVersion, modelDB.Version{
		UpdatedAt: time.Now(),
		Status:    modelPB.ModelVersion_STATUS_OFFLINE.String(),
	}); err != nil {
		return err
	}

	return nil
}

func (s *modelService) CreateModel(model *modelDB.Model) (*modelDB.Model, error) {
	// Validate the naming rule of model
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", model.Name); !match {
		return &modelDB.Model{}, status.Error(codes.FailedPrecondition, "The name of model is invalid")
	}

	if existingModel, _ := s.GetModelByName(model.Namespace, model.Name); existingModel.Name != "" {
		return &modelDB.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", model.Name)
	}

	if err := s.modelRepository.CreateModel(*model); err != nil {
		return &modelDB.Model{}, err
	}

	if createdModel, err := s.GetModelByName(model.Namespace, model.Name); err != nil {
		return &modelDB.Model{}, err
	} else {
		return &createdModel, nil
	}
}

func (s *modelService) GetModelByName(namespace string, modelName string) (modelDB.Model, error) {
	return s.modelRepository.GetModelByName(namespace, modelName)
}

func (s *modelService) GetModelVersionLatest(modelId uint64) (modelDB.Version, error) {
	return s.modelRepository.GetModelVersionLatest(modelId)
}

func (s *modelService) CreateVersion(version modelDB.Version) (modelDB.Version, error) {
	if err := s.modelRepository.CreateVersion(version); err != nil {
		return modelDB.Version{}, err
	}

	if createdVersion, err := s.modelRepository.GetModelVersion(version.ModelId, version.Version); err != nil {
		return modelDB.Version{}, err
	} else {
		return createdVersion, nil
	}
}

func (s *modelService) GetModelVersion(modelId uint64, version uint64) (modelDB.Version, error) {
	return s.modelRepository.GetModelVersion(modelId, version)
}

func (s *modelService) GetModelVersions(modelId uint64) ([]modelDB.Version, error) {
	return s.modelRepository.GetModelVersions(modelId)
}

func (s *modelService) GetTModels(modelId uint64) ([]modelDB.TModel, error) {
	return s.modelRepository.GetTModels(modelId)
}

func (s *modelService) ModelInfer(namespace string, modelName string, version uint64, imgsBytes [][]byte, task modelPB.Model_Task) (interface{}, error) {
	// Triton model name is change into
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return nil, fmt.Errorf("Model not found")
	}

	ensembleModel, err := s.modelRepository.GetTEnsembleModel(modelInDB.Id, version)
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
	case modelPB.Model_TASK_CLASSIFICATION:
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

	case modelPB.Model_TASK_DETECTION:
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

func createModel(s *modelService, namespace string, uploadedModel *modelDB.Model) (modelDB.Model, []modelDB.Version, []modelDB.TModel, error) {

	modelInDB, err := s.GetModelByName(namespace, uploadedModel.Name)
	if err != nil {
		createdModel, err := s.CreateModel(uploadedModel)
		if err != nil {
			return modelDB.Model{}, []modelDB.Version{}, []modelDB.TModel{}, fmt.Errorf("Could not create model in DB")
		}
		modelInDB = *createdModel
	}
	latestVersion, err := s.modelRepository.GetModelVersionLatest(modelInDB.Id)
	if err == nil {
		uploadedModel.Versions[0].Version = latestVersion.Version + 1
	} else {
		uploadedModel.Versions[0].Version = 1
	}
	uploadedModel.Versions[0].ModelId = modelInDB.Id
	versionInDB, err := s.CreateVersion(uploadedModel.Versions[0])
	if err != nil {
		return modelDB.Model{}, []modelDB.Version{}, []modelDB.TModel{}, fmt.Errorf("Could not create model version in DB")
	}
	for i := 0; i < len(uploadedModel.TritonModels); i++ {
		tritonModel := uploadedModel.TritonModels[i]
		tritonModel.ModelId = modelInDB.Id
		tritonModel.ModelVersion = versionInDB.Version
		err = s.modelRepository.CreateTModel(tritonModel)
		if err != nil {
			return modelDB.Model{}, []modelDB.Version{}, []modelDB.TModel{}, fmt.Errorf("Could not create triton model in DB")
		}
	}
	versions, err := s.GetModelVersions(modelInDB.Id)
	if err != nil {
		return modelDB.Model{}, []modelDB.Version{}, []modelDB.TModel{}, fmt.Errorf("Could not get model versions in DB")
	}

	return modelInDB, versions, uploadedModel.TritonModels, nil
}

func (s *modelService) CreateModelBinaryFileUpload(namespace string, uploadedModel *modelDB.Model) (*modelPB.Model, error) {
	modelInDB, versions, tritonModels, err := createModel(s, namespace, uploadedModel)
	return createModelInfo(modelInDB, versions, tritonModels), err
}

func (s *modelService) ListModels(namespace string) ([]*modelPB.Model, error) {
	models, err := s.modelRepository.ListModels(modelDB.ListModelQuery{Namespace: namespace})
	if err != nil {
		return []*modelPB.Model{}, err
	}

	var resModels []*modelPB.Model
	for i := 0; i < len(models); i++ {
		md := models[i]
		versions, err := s.GetModelVersions(md.Id)
		if err != nil {
			return []*modelPB.Model{}, err
		}
		tritonModels, err := s.GetTModels(md.Id)
		if err != nil {
			return []*modelPB.Model{}, err

		}
		resModels = append(resModels, createModelInfo(md, versions, tritonModels))
	}

	return resModels, nil
}

func (s *modelService) UpdateModelVersion(namespace string, in *modelPB.UpdateModelVersionRequest) (*modelPB.ModelVersion, error) {
	modelInDB, err := s.GetModelByName(namespace, in.Name)
	if err != nil {
		return &modelPB.ModelVersion{}, err
	}
	if in.FieldMask != nil && len(in.FieldMask.Paths) > 0 {
		for _, field := range in.FieldMask.Paths {
			switch field {
			case "status":
				switch in.VersionPatch.Status {
				case modelPB.ModelVersion_STATUS_ONLINE:
					if err := setModelOnline(s, modelInDB.Id, in.Version); err != nil {
						return &modelPB.ModelVersion{}, err
					}
				case modelPB.ModelVersion_STATUS_OFFLINE:
					if err := setModelOffline(s, modelInDB.Id, in.Version); err != nil {
						return &modelPB.ModelVersion{}, err
					}
				default:
					return &modelPB.ModelVersion{}, fmt.Errorf("Wrong status value. Status should be ONLINE or OFFLINE")
				}
			case "description":
				err = s.modelRepository.UpdateModelVersion(modelInDB.Id, in.Version, modelDB.Version{
					UpdatedAt:   time.Now(),
					Description: in.VersionPatch.Description,
				})
				if err != nil {
					return &modelPB.ModelVersion{}, err
				}
			}
		}
	}
	modelVersionInDB, err := s.GetModelVersion(modelInDB.Id, in.Version)
	return createModelVersion(modelVersionInDB), err
}

func (s *modelService) GetFullModelData(namespace string, modelName string) (*modelPB.Model, error) {
	// TODO: improve by using join
	resModelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return &modelPB.Model{}, err
	}

	versions, err := s.GetModelVersions(resModelInDB.Id)
	if err != nil {
		return &modelPB.Model{}, err
	}

	tritonModels, err := s.GetTModels(resModelInDB.Id)
	if err != nil {
		return &modelPB.Model{}, err
	}

	return createModelInfo(resModelInDB, versions, tritonModels), nil
}

func (s *modelService) DeleteModel(namespace string, modelName string) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelVersionsInDB, err := s.GetModelVersions(modelInDB.Id)
	if err == nil {
		for i := 0; i < len(modelVersionsInDB); i++ {
			if err := setModelOffline(s, modelInDB.Id, modelVersionsInDB[i].Version); err != nil {
				return err
			}
		}
		tritonModels, err := s.modelRepository.GetTModels(modelInDB.Id)
		if err == nil {
			for i := 0; i < len(tritonModels); i++ {
				modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
				_ = os.RemoveAll(modelDir)
			}
		}
	}

	return s.modelRepository.DeleteModel(modelInDB.Id)
}

func (s *modelService) DeleteModelVersion(namespace string, modelName string, version uint64) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelVersionInDB, err := s.GetModelVersion(modelInDB.Id, version)
	if err != nil {
		return err
	}

	if err := setModelOffline(s, modelInDB.Id, modelVersionInDB.Version); err != nil {
		return err
	}

	tritonModels, err := s.modelRepository.GetTModelVersions(modelInDB.Id, modelVersionInDB.Version)
	if err == nil {
		for i := 0; i < len(tritonModels); i++ {
			modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
			_ = os.RemoveAll(modelDir)
		}
	}

	modelVersionsInDB, err := s.GetModelVersions(modelInDB.Id)
	if err != nil {
		return err
	}

	if len(modelVersionsInDB) > 1 {
		return s.modelRepository.DeleteModelVersion(modelInDB.Id, modelVersionInDB.Version)
	} else {
		return s.modelRepository.DeleteModel(modelInDB.Id)
	}
}
