package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/models"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/protogen-go/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeError(statusCode codes.Code, title string, detail string) error {
	err := &models.Error{
		Status: int32(statusCode),
		Title:  title,
		Detail: detail,
	}
	data, _ := json.Marshal(err)
	return status.Error(statusCode, string(data))
}

func createModelResponse(modelInDB models.Model, versions []models.Version, tritonModels []models.TModel) *models.ModelResponse {
	var vers []models.VersionResponse
	for i := 0; i < len(versions); i++ {
		vers = append(vers, models.VersionResponse{
			Version:     versions[i].Version,
			ModelId:     versions[i].ModelId,
			Description: versions[i].Description,
			CreatedAt:   versions[i].CreatedAt,
			UpdatedAt:   versions[i].UpdatedAt,
			Status:      versions[i].Status,
		})
	}
	var contents []string
	for i := 0; i < len(tritonModels); i++ {
		if len(strings.Split(tritonModels[i].Name, "#")) > 2 {
			contents = append(contents, strings.Split(tritonModels[i].Name, "#")[2])
		}
	}
	return &models.ModelResponse{
		Name:       modelInDB.Name,
		FullName:   modelInDB.FullName,
		Id:         modelInDB.Id,
		Optimized:  modelInDB.Optimized,
		Framework:  modelInDB.Framework,
		Type:       modelInDB.Type,
		Visibility: modelInDB.Visibility,
		CVTask:     modelInDB.CVTask,
		Versions:   vers,
		Contents:   contents,
	}
}

func createModelInfo(modelInDB models.Model, versions []models.Version, tritonModels []models.TModel) *model.ModelInfo {
	var vers []*model.ModelVersion
	for i := 0; i < len(versions); i++ {
		ver := versions[i]
		var st = model.ModelStatus_OFFLINE
		if ver.Status == model.ModelStatus_ONLINE.String() {
			st = model.ModelStatus_ONLINE
		} else if ver.Status == model.ModelStatus_ERROR.String() {
			st = model.ModelStatus_ERROR
		}

		vers = append(vers, &model.ModelVersion{
			Version:     ver.Version,
			ModelId:     ver.ModelId,
			Description: ver.Description,
			CreatedAt:   timestamppb.New(ver.CreatedAt),
			UpdatedAt:   timestamppb.New(ver.UpdatedAt),
			Status:      st,
		})
	}
	var contents []string
	for i := 0; i < len(tritonModels); i++ {
		if len(strings.Split(tritonModels[i].Name, "#")) > 2 {
			contents = append(contents, strings.Split(tritonModels[i].Name, "#")[2])
		}
	}
	return &model.ModelInfo{
		Name:       modelInDB.Name,
		FullName:   modelInDB.FullName,
		Id:         modelInDB.Id,
		Optimized:  modelInDB.Optimized,
		Framework:  modelInDB.Framework,
		Type:       modelInDB.Type,
		Visibility: modelInDB.Visibility,
		CvTask:     model.CVTask(modelInDB.CVTask),
		Versions:   vers,
		Contents:   contents,
	}
}

type ModelService interface {
	CreateModel(model *models.Model) (*models.Model, error)
	GetModelByName(namespace string, modelName string) (models.Model, error)
	GetModelMetaData(namespace string, modelName string) (*model.ModelInfo, error)
	CreateVersion(version models.Version) (models.Version, error)
	UpdateModelVersions(modelId int32, version models.Version) error
	UpdateModelMetaData(namespace string, updatedModel models.Model) (*model.ModelInfo, error)
	GetModelVersion(modelId int32, version int32) (models.Version, error)
	GetModelVersions(modelId int32) ([]models.Version, error)
	PredictModelByUpload(namespace string, modelName string, version int32, filePath string, cvTask model.CVTask) (interface{}, error)
	CreateModelByUpload(namespace string, createdModel *models.Model) (*model.ModelInfo, error)
	HandleCreateModelByUpload(namespace string, createdModel *models.Model) (*models.ModelResponse, error)
	ListModels(namespace string) ([]*model.ModelInfo, error)
	UpdateModel(namespace string, updatedInfo *model.UpdateModelRequest) (*model.ModelInfo, error)
	DeleteModel(namespace string, modelName string) error
}

type modelService struct {
	modelRepository repository.ModelRepository
}

func NewModelService(r repository.ModelRepository) ModelService {
	return &modelService{
		modelRepository: r,
	}
}

func (s *modelService) CreateModel(model *models.Model) (*models.Model, error) {
	// Validate the naming rule of model
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", model.Name); !match {
		return &models.Model{}, status.Error(codes.FailedPrecondition, "The name of model is invalid")
	}

	if existingModel, _ := s.GetModelByName(model.Namespace, model.Name); existingModel.Name != "" {
		return &models.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", model.Name)
	}

	if err := s.modelRepository.CreateModel(*model); err != nil {
		return &models.Model{}, err
	}

	if createdModel, err := s.GetModelByName(model.Namespace, model.Name); err != nil {
		return &models.Model{}, err
	} else {
		return &createdModel, nil
	}
}

func (s *modelService) GetModelByName(namespace string, modelName string) (models.Model, error) {
	return s.modelRepository.GetModelByName(namespace, modelName)
}

func (s *modelService) CreateVersion(version models.Version) (models.Version, error) {
	if err := s.modelRepository.CreateVersion(version); err != nil {
		return models.Version{}, err
	}

	if createdVersion, err := s.modelRepository.GetModelVersion(version.ModelId, version.Version); err != nil {
		return models.Version{}, err
	} else {
		return createdVersion, nil
	}
}

func (s *modelService) UpdateModelVersions(modelId int32, version models.Version) error {
	return s.modelRepository.UpdateModelVersions(modelId, version)
}

func (s *modelService) GetModelVersion(modelId int32, version int32) (models.Version, error) {
	return s.modelRepository.GetModelVersion(modelId, version)
}

func (s *modelService) GetModelVersions(modelId int32) ([]models.Version, error) {
	return s.modelRepository.GetModelVersions(modelId)
}

func (s *modelService) GetTModels(modelId int32) ([]models.TModel, error) {
	return s.modelRepository.GetTModels(modelId)
}

func (s *modelService) PredictModelByUpload(namespace string, modelName string, version int32, filePath string, cvTask model.CVTask) (interface{}, error) {
	// Triton model name is change into
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return nil, makeError(400, "PredictModel", "Model not found")
	}

	ensembleModel, err := s.modelRepository.GetTEnsembleModel(modelInDB.Id)
	if err != nil {
		return nil, makeError(400, "PredictModel", "Ensemble model not found")
	}

	ensembleModelName := ensembleModel.Name
	modelMetadataResponse := triton.ModelMetadataRequest(triton.TritonClient, ensembleModelName, fmt.Sprint(version))
	if modelMetadataResponse == nil {
		return nil, makeError(400, "PredictModel", "Model is offline")
	}
	modelConfigResponse := triton.ModelConfigRequest(triton.TritonClient, ensembleModelName, fmt.Sprint(version))
	if modelMetadataResponse == nil {
		return nil, makeError(400, "PredictModel", "Model config not found")
	}
	input, err := triton.PreProcess(filePath, modelMetadataResponse, modelConfigResponse, cvTask)
	if err != nil {
		return nil, makeError(400, "PredictModel", err.Error())
	}
	// /* We use a simple model that takes 2 input tensors of 16 integers
	// each and returns 2 output tensors of 16 integers each. One
	// output tensor is the element-wise sum of the inputs and one
	// output is the element-wise difference. */
	inferResponse, err := triton.ModelInferRequest(triton.TritonClient, cvTask, input, ensembleModelName, fmt.Sprint(version), modelMetadataResponse, modelConfigResponse)
	if err != nil {
		return nil, makeError(500, "PredictModel InferRequest", err.Error())
	}
	// /* We expect there to be 2 results (each with batch-size 1). Walk
	// over all 16 result elements and print the sum and difference
	// calculated by the model. */
	postprocessResponse, err := triton.PostProcess(inferResponse, modelMetadataResponse, cvTask)
	if err != nil {
		return nil, makeError(500, "PredictModel PostProcess", err.Error())
	}
	switch cvTask {
	case model.CVTask_CLASSIFICATION:
		clsResponses := postprocessResponse.([]string)
		var contents []*model.ClassificationOutput
		for _, clsRes := range clsResponses {
			clsResSplit := strings.Split(clsRes, ":")
			if len(clsResSplit) != 3 {
				return nil, makeError(500, "PredictModel", "Unable to decode inference output")
			}
			score, err := strconv.ParseFloat(clsResSplit[0], 32)
			if err != nil {
				return nil, makeError(500, "PredictModel", "Unable to decode inference output")
			}
			clsOutput := model.ClassificationOutput{
				Category: clsResSplit[2],
				Score:    float32(score),
			}
			contents = append(contents, &clsOutput)
		}
		clsOutputs := model.ClassificationOutputs{
			Contents: contents,
		}
		return &clsOutputs, nil

	case model.CVTask_DETECTION:
		detResponses := postprocessResponse.(triton.DetectionOutput)
		batchedOutputDataBboxes := detResponses.Boxes
		batchedOutputDataLabels := detResponses.Labels
		var detOutputs model.DetectionOutputs
		for i := range batchedOutputDataBboxes {
			var contents []*model.BoundingBoxPrediction
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]

				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Triton to be able to batch Tensors
				if label != "0" {
					pred := &model.BoundingBoxPrediction{
						Category: label,
						Score:    box[4],
						// Convert x1y1x2y2 to xywh where xy is top-left corner
						Box: &model.Box{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2] - box[0],
							Height: box[3] - box[1],
						},
					}
					contents = append(contents, pred)
				}
			}
			detOutput := &model.DetectionOutput{
				Contents: contents,
			}
			detOutputs.Contents = append(detOutputs.Contents, detOutput)
		}
		return &detOutputs, nil
	default:
		return postprocessResponse, nil
	}
}

func createModel(s *modelService, namespace string, uploadedModel *models.Model) (models.Model, []models.Version, []models.TModel, error) {
	modelInDB, err := s.GetModelByName(namespace, uploadedModel.Name)
	if err != nil {
		createdModel, err := s.CreateModel(uploadedModel)
		if err != nil {
			return models.Model{}, []models.Version{}, []models.TModel{}, makeError(500, "CreateModel", "Could not create model in DB")
		}
		modelInDB = *createdModel
	}

	versionInDB, err := s.GetModelVersion(modelInDB.Id, uploadedModel.Versions[0].Version)
	if err == nil {
		return models.Model{}, []models.Version{}, []models.TModel{}, makeError(409, "CreateModel", fmt.Sprintf("Model with name %v and version %v already existed", uploadedModel.Name, versionInDB.Version))
	}

	uploadedModel.Versions[0].ModelId = modelInDB.Id
	_, err = s.CreateVersion(uploadedModel.Versions[0])
	if err != nil {
		return models.Model{}, []models.Version{}, []models.TModel{}, makeError(500, "CreateModel", "Could not create model version in DB")
	}
	for i := 0; i < len(uploadedModel.TritonModels); i++ {
		tmodel := uploadedModel.TritonModels[i]
		tmodel.ModelId = modelInDB.Id
		err = s.modelRepository.CreateTModel(tmodel)
		if err != nil {
			return models.Model{}, []models.Version{}, []models.TModel{}, makeError(500, "CreateModel", "Could not create triton model in DB")
		}
	}
	versions, err := s.GetModelVersions(modelInDB.Id)
	if err != nil {
		return models.Model{}, []models.Version{}, []models.TModel{}, makeError(500, "CreateModel", "Could not get model versions in DB")
	}

	return modelInDB, versions, uploadedModel.TritonModels, nil
}

func (s *modelService) CreateModelByUpload(namespace string, uploadedModel *models.Model) (*model.ModelInfo, error) {
	modelInDB, versions, tritonModels, err := createModel(s, namespace, uploadedModel)
	return createModelInfo(modelInDB, versions, tritonModels), err
}

func (s *modelService) HandleCreateModelByUpload(namespace string, uploadedModel *models.Model) (*models.ModelResponse, error) {
	modelInDB, versions, tritonModels, err := createModel(s, namespace, uploadedModel)
	return createModelResponse(modelInDB, versions, tritonModels), err
}

func (s *modelService) ListModels(namespace string) ([]*model.ModelInfo, error) {
	if !triton.IsTritonServerReady() {
		return []*model.ModelInfo{}, makeError(503, "ListModels", "Triton Server not ready yet")
	}

	models, err := s.modelRepository.ListModels(models.ListModelQuery{Namespace: namespace})
	if err != nil {
		return []*model.ModelInfo{}, makeError(500, "ListModels", err.Error())
	}

	var resModels []*model.ModelInfo
	for i := 0; i < len(models); i++ {
		md := models[i]
		versions, err := s.GetModelVersions(md.Id)
		if err != nil {
			return []*model.ModelInfo{}, makeError(500, "ListModels", "Could not get model versions in DB")
		}
		tritonModels, err := s.GetTModels(md.Id)
		if err != nil {
			return []*model.ModelInfo{}, makeError(500, "ListModels", "Could not get triton model in DB")

		}
		resModels = append(resModels, createModelInfo(md, versions, tritonModels))
	}

	return resModels, nil
}

func (s *modelService) UpdateModelMetaData(namespace string, updatedModel models.Model) (*model.ModelInfo, error) {
	md, err := s.GetModelByName(namespace, updatedModel.Name)
	if err != nil {
		return &model.ModelInfo{}, makeError(404, "UpdateModelMetaData Error", "The model not found in server")
	}

	err = s.modelRepository.UpdateModelMetaData(md.Id, updatedModel)
	if err != nil {
		return &model.ModelInfo{}, makeError(500, "UpdateModelMetaData Error", err.Error())
	}

	modelInfo, err := s.GetModelMetaData(namespace, md.Name)
	if err != nil {
		return &model.ModelInfo{}, makeError(500, "UpdateModelMetaData Error", err.Error())
	}

	return modelInfo, nil
}

func (s *modelService) UpdateModel(namespace string, in *model.UpdateModelRequest) (*model.ModelInfo, error) {
	modelInDB, err := s.GetModelByName(namespace, in.Model.Name)
	if err != nil {
		return &model.ModelInfo{}, makeError(404, "UpdateModel Error", "The model not found in server")
	}

	if in.UpdateMask != nil && len(in.UpdateMask.Paths) > 0 {
		for _, field := range in.UpdateMask.Paths {
			switch field {
			case "status":
				ensembleModel, err := s.modelRepository.GetTEnsembleModel(modelInDB.Id)
				if err != nil {
					return &model.ModelInfo{}, makeError(404, "UpdateModel Error", "Could not find ensemble model")
				}
				switch in.Model.Status {
				case model.ModelStatus_ONLINE:
					_, err = triton.LoadModelRequest(triton.TritonClient, ensembleModel.Name)
					if err != nil {
						err = s.UpdateModelVersions(modelInDB.Id, models.Version{
							UpdatedAt: time.Now(),
							Status:    model.ModelStatus_ERROR.String(),
						})
						if err != nil {
							return &model.ModelInfo{}, makeError(500, "UpdateModel Error", "Could not update model status")
						}
					} else {
						err := s.UpdateModelVersions(modelInDB.Id, models.Version{
							UpdatedAt: time.Now(),
							Status:    model.ModelStatus_ONLINE.String(),
						})
						if err != nil {
							return &model.ModelInfo{}, makeError(500, "UpdateModel Error", "Could not update model version status")
						}
					}
				case model.ModelStatus_OFFLINE:
					_, err = triton.UnloadModelRequest(triton.TritonClient, ensembleModel.Name)
					if err != nil {
						err = s.UpdateModelVersions(modelInDB.Id, models.Version{
							UpdatedAt: time.Now(),
							Status:    model.ModelStatus_ERROR.String(),
						})
						if err != nil {
							return &model.ModelInfo{}, makeError(500, "UpdateModel Error", "Could not update model status")
						}
					} else {
						err = s.UpdateModelVersions(modelInDB.Id, models.Version{
							UpdatedAt: time.Now(),
							Status:    model.ModelStatus_OFFLINE.String(),
						})
						if err != nil {
							return &model.ModelInfo{}, makeError(500, "UpdateModel Error", "Could not update model status")
						}
					}
				default:
					return &model.ModelInfo{}, makeError(400, "UpdateModel Error", "Wrong status value. Status should be online or offline")
				}
			}
		}
	}
	return s.GetModelMetaData(namespace, in.Model.Name)
}

func (s *modelService) GetModelMetaData(namespace string, modelName string) (*model.ModelInfo, error) {
	// TODO: improve by using join
	resModelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return &model.ModelInfo{}, makeError(404, "GetModel", fmt.Sprintf("Model %v not found in namespace %v", modelName, namespace))
	}

	versions, err := s.GetModelVersions(resModelInDB.Id)
	if err != nil {
		return &model.ModelInfo{}, makeError(404, "GetModel", "There is no versions for this model")
	}

	tritonModels, err := s.GetTModels(resModelInDB.Id)
	if err != nil {
		return &model.ModelInfo{}, makeError(404, "GetModel", "There is no triton model for this model")
	}

	return createModelInfo(resModelInDB, versions, tritonModels), nil
}

func (s *modelService) DeleteModel(namespace string, modelName string) error {
	resModelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return makeError(404, "DeleteModel", fmt.Sprintf("Model %v not found in namespace %v", modelName, namespace))
	}
	return s.modelRepository.DeleteModel(resModelInDB.Id)
}
