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
	"github.com/instill-ai/model-backend/protogen-go/model"
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

func createModelResponse(modelInDB models.Model, versions []models.Version) *models.ModelResponse {
	var mRes models.ModelResponse
	mRes.Name = modelInDB.Name
	mRes.FullName = modelInDB.FullName
	mRes.Id = modelInDB.Id
	mRes.Optimized = modelInDB.Optimized
	mRes.Description = modelInDB.Description
	mRes.Framework = modelInDB.Framework
	mRes.CreatedAt = modelInDB.CreatedAt
	mRes.UpdatedAt = modelInDB.UpdatedAt
	mRes.Organization = modelInDB.Organization
	mRes.Icon = modelInDB.Icon
	mRes.Type = modelInDB.Type
	mRes.Visibility = modelInDB.Visibility
	var vers []models.VersionResponse
	for i := 0; i < len(versions); i++ {
		vers = append(vers, models.VersionResponse{
			Version:     versions[i].Version,
			ModelId:     versions[i].ModelId,
			Description: versions[i].Description,
			CreatedAt:   versions[i].CreatedAt,
			UpdatedAt:   versions[i].UpdatedAt,
		})
	}
	mRes.Versions = vers
	return &mRes
}

func createModelResponseByUpload(modelInDB models.Model, versions []models.Version) *model.CreateModelResponse {
	var mRes model.CreateModelResponse
	mRes.Name = modelInDB.Name
	mRes.FullName = modelInDB.FullName
	mRes.Id = modelInDB.Id
	mRes.Optimized = modelInDB.Optimized
	mRes.Description = modelInDB.Description
	mRes.Framework = modelInDB.Framework
	mRes.CreatedAt = timestamppb.New(modelInDB.CreatedAt)
	mRes.UpdatedAt = timestamppb.New(modelInDB.UpdatedAt)
	mRes.Organization = modelInDB.Organization
	mRes.Icon = modelInDB.Icon
	mRes.Type = modelInDB.Type
	mRes.Visibility = modelInDB.Visibility
	var vers []*model.ModelVersion
	for i := 0; i < len(versions); i++ {
		vers = append(vers, &model.ModelVersion{
			Version:     versions[i].Version,
			ModelId:     versions[i].ModelId,
			Description: versions[i].Description,
			CreatedAt:   timestamppb.New(versions[i].CreatedAt),
			UpdatedAt:   timestamppb.New(versions[i].UpdatedAt),
		})
	}
	mRes.Versions = vers
	return &mRes
}

type ModelService interface {
	CreateModel(model models.Model) (models.Model, error)
	GetModelByName(namespace string, modelName string) (models.Model, error)
	CreateVersion(version models.Version) (models.Version, error)
	GetVersionByModel(modelId int32, version int32) (models.Version, error)
	PredictModelByUpload(namespace string, modelName string, version int32, filePath string, cvTask triton.CVTask) (interface{}, error)
	CreateModelByUpload(namespace string, createdModels []*models.Model, createdVersions []*models.Version) ([]*model.CreateModelResponse, error)
	HandleCreateModelByUpload(namespace string, createdModels []*models.Model, createdVersions []*models.Version) ([]*models.ModelResponse, error)
	ListModels(username string) ([]*model.CreateModelResponse, error)
}

type modelService struct {
	modelRepository repository.ModelRepository
}

func NewModelService(r repository.ModelRepository) ModelService {
	return &modelService{
		modelRepository: r,
	}
}

func (s *modelService) CreateModel(model models.Model) (models.Model, error) {
	// Validate the naming rule of model
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", model.Name); !match {
		return models.Model{}, status.Error(codes.FailedPrecondition, "The name of model is invalid")
	}

	if existingModel, _ := s.GetModelByName(model.Namespace, model.Name); existingModel.Name != "" {
		return models.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", model.Name)
	}

	if err := s.modelRepository.CreateModel(model); err != nil {
		return models.Model{}, err
	}

	if createdModel, err := s.GetModelByName(model.Namespace, model.Name); err != nil {
		return models.Model{}, err
	} else {
		return createdModel, nil
	}
}

func (s *modelService) GetModelByName(namespace string, modelName string) (models.Model, error) {
	return s.modelRepository.GetModelByName(namespace, modelName)
}

func (s *modelService) CreateVersion(version models.Version) (models.Version, error) {
	if err := s.modelRepository.CreateVersion(version); err != nil {
		return models.Version{}, err
	}

	if createdVersion, err := s.modelRepository.GetVersionByModel(version.ModelId, version.Version); err != nil {
		return models.Version{}, err
	} else {
		return createdVersion, nil
	}
}

func (s *modelService) GetVersionByModel(modelId int32, version int32) (models.Version, error) {
	return s.modelRepository.GetVersionByModel(modelId, version)
}

func (s *modelService) PredictModelByUpload(namespace string, modelName string, version int32, filePath string, cvTask triton.CVTask) (interface{}, error) {
	modelMetadataResponse := triton.ModelMetadataRequest(triton.TritonClient, modelName, fmt.Sprint(version))
	if modelMetadataResponse == nil {
		return nil, makeError(400, "PredictModel", "Model not found")
	}

	modelConfigResponse := triton.ModelConfigRequest(triton.TritonClient, modelName, fmt.Sprint(version))
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
	inferResponse, err := triton.ModelInferRequest(triton.TritonClient, cvTask, input, modelName, fmt.Sprint(version), modelMetadataResponse, modelConfigResponse)
	if err != nil {
		return nil, makeError(500, "PredictModel", err.Error())
	}

	// /* We expect there to be 2 results (each with batch-size 1). Walk
	// over all 16 result elements and print the sum and difference
	// calculated by the model. */
	postprocessResponse, err := triton.PostProcess(inferResponse, modelMetadataResponse, cvTask)
	if err != nil {
		return nil, makeError(500, "PredictModel", err.Error())
	}

	switch cvTask {
	case triton.Classification:
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

	case triton.Detection:
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
		return nil, makeError(500, "PredictModel", fmt.Sprintf("modelType %v do not support", cvTask))
	}
}

func (s *modelService) CreateModelByUpload(namespace string, createdModels []*models.Model, createdVersions []*models.Version) ([]*model.CreateModelResponse, error) {
	var respModels = []*model.CreateModelResponse{}
	for i := 0; i < len(createdModels); i++ {
		newModel := createdModels[i]
		// check model existed or not
		var versions []models.Version
		modelInDB, err := s.GetModelByName(namespace, newModel.Name)
		if err == nil { // model already existed
			// check version exited or not
			for j := 0; j < len(createdVersions); j++ { // this list contain versions of all models, so need check model id; TODO: maybe use bidirection link in DB
				if createdVersions[j].ModelName != modelInDB.Name {
					continue
				}
				newVersion, err := s.CreateVersion(models.Version{
					Version:     createdVersions[j].Version,
					ModelId:     modelInDB.Id,
					Description: modelInDB.Description,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Status:      "offline",
					Metadata:    models.JSONB{},
				})
				if err == nil { // new version created
					versions = append(versions, newVersion)
				} else { // version already existed
					versionInDB, err := s.GetVersionByModel(modelInDB.Id, createdVersions[j].Version)
					if err == nil {
						versions = append(versions, versionInDB)
					}
				}
			}
			respModel := createModelResponseByUpload(modelInDB, versions)
			respModels = append(respModels, respModel)
			continue
		}

		newModel.Author = namespace
		newModel.Namespace = namespace
		_, err = s.CreateModel(*newModel)
		if err != nil {
			continue
		}

		for j := 0; j < len(createdVersions); j++ {
			if createdVersions[j].ModelName != modelInDB.Name {
				continue
			}
			newVersion, err := s.CreateVersion(models.Version{
				Version:     createdVersions[j].Version,
				ModelId:     modelInDB.Id,
				Description: modelInDB.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      "offline",
				Metadata:    models.JSONB{},
			})
			if err == nil { // new version created
				versions = append(versions, newVersion)
			} else { // version already existed
				versionInDB, err := s.GetVersionByModel(modelInDB.Id, createdVersions[j].Version)
				if err == nil {
					versions = append(versions, versionInDB)
				}
			}
		}

		modelInDB, err = s.GetModelByName(namespace, newModel.Name)
		if err != nil {
			return nil, err
		}

		respModel := createModelResponseByUpload(modelInDB, versions)
		respModels = append(respModels, respModel)
	}

	return respModels, nil
}

func (s *modelService) HandleCreateModelByUpload(namespace string, createdModels []*models.Model, createdVersions []*models.Version) ([]*models.ModelResponse, error) {
	var respModels = []*models.ModelResponse{}
	for i := 0; i < len(createdModels); i++ {
		newModel := createdModels[i]
		// check model existed or not
		var versions []models.Version
		modelInDB, err := s.GetModelByName(namespace, newModel.Name)
		if err == nil { // model already existed
			// check version exited or not
			for j := 0; j < len(createdVersions); j++ { // this list contain versions of all models, so need check model id; TODO: maybe use bidirection link in DB
				if createdVersions[j].ModelName != modelInDB.Name {
					continue
				}
				newVersion, err := s.CreateVersion(models.Version{
					Version:     createdVersions[j].Version,
					ModelId:     modelInDB.Id,
					Description: modelInDB.Description,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Status:      "offline",
					Metadata:    models.JSONB{},
				})
				if err == nil { // new version created
					versions = append(versions, newVersion)
				} else { // version already existed
					versionInDB, err := s.GetVersionByModel(modelInDB.Id, createdVersions[j].Version)
					if err == nil {
						versions = append(versions, versionInDB)
					}
				}
			}
			respModel := createModelResponse(modelInDB, versions)
			respModels = append(respModels, respModel)
			continue
		}

		newModel.Author = namespace
		newModel.Namespace = namespace
		_, err = s.CreateModel(*newModel)
		if err != nil {
			continue
		}

		for j := 0; j < len(createdVersions); j++ {
			if createdVersions[j].ModelName != modelInDB.Name {
				continue
			}
			newVersion, err := s.CreateVersion(models.Version{
				Version:     createdVersions[j].Version,
				ModelId:     modelInDB.Id,
				Description: modelInDB.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      "offline",
				Metadata:    models.JSONB{},
			})
			if err == nil { // new version created
				versions = append(versions, newVersion)
			} else { // version already existed
				versionInDB, err := s.GetVersionByModel(modelInDB.Id, createdVersions[j].Version)
				if err == nil {
					versions = append(versions, versionInDB)
				}
			}
		}

		modelInDB, err = s.GetModelByName(namespace, newModel.Name)
		if err != nil {
			return nil, err
		}

		respModel := createModelResponse(modelInDB, versions)
		respModels = append(respModels, respModel)
	}

	return respModels, nil
}

func (s *modelService) ListModels(username string) ([]*model.CreateModelResponse, error) {
	if !triton.IsTritonServerReady() {
		return []*model.CreateModelResponse{}, makeError(503, "PredictModel", "Triton Server not ready yet")
	}

	listModelsResponse := triton.ListModelsRequest(triton.TritonClient)

	var resModels []*model.CreateModelResponse
	models := listModelsResponse.Models
	for i := 0; i < len(models); i++ {
		md, err := s.GetModelByName(username, models[i].Name)
		if err == nil {
			resModels = append(resModels, &model.CreateModelResponse{
				Id:          md.Id,
				Name:        md.Name,
				Optimized:   md.Optimized,
				Description: md.Description,
				Type:        md.Type,
				Framework:   md.Framework,
				Author:      md.Author,
				Icon:        md.Icon,
			})
		}
	}

	return resModels, nil
}
