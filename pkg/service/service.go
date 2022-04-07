package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

type Service interface {
	CreateModel(model *datamodel.Model) (*datamodel.Model, error)
	GetModelByName(namespace string, modelName string) (datamodel.Model, error)
	CreateVersion(version datamodel.Version) (datamodel.Version, error)
	GetModelVersion(modelId uint, version uint) (datamodel.Version, error)
	GetModelVersions(modelId uint) ([]datamodel.Version, error)
	GetModelVersionLatest(modelId uint) (datamodel.Version, error)
	GetFullModelData(namespace string, modelName string) (*modelPB.Model, error)
	ModelInfer(namespace string, modelName string, version uint, imgsBytes [][]byte, task modelPB.Model_Task) (interface{}, error)
	CreateModelBinaryFileUpload(namespace string, createdModel *datamodel.Model) (*modelPB.Model, error)
	ListModels(namespace string) ([]*modelPB.Model, error)
	UpdateModelVersion(namespace string, updatedInfo *modelPB.UpdateModelVersionRequest) (*modelPB.ModelVersion, error)
	DeleteModel(namespace string, modelName string) error
	DeleteModelVersion(namespace string, modelName string, version uint) error
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

func createModelVersion(modelVersionInDB datamodel.Version) *modelPB.ModelVersion {
	var st = modelPB.ModelVersion_STATUS_OFFLINE
	if string(modelVersionInDB.Status) == modelPB.ModelVersion_STATUS_ONLINE.String() {
		st = modelPB.ModelVersion_STATUS_ONLINE
	} else if string(modelVersionInDB.Status) == modelPB.ModelVersion_STATUS_ERROR.String() {
		st = modelPB.ModelVersion_STATUS_ERROR
	}

	var gitRef modelPB.GitRef
	if modelVersionInDB.Github.GitRef.Branch != "" {
		gitRef = modelPB.GitRef{
			Ref: &modelPB.GitRef_Branch{
				Branch: modelVersionInDB.Github.GitRef.Branch,
			},
		}
	} else if modelVersionInDB.Github.GitRef.Tag != "" {
		gitRef = modelPB.GitRef{
			Ref: &modelPB.GitRef_Tag{
				Tag: modelVersionInDB.Github.GitRef.Tag,
			},
		}
	} else if modelVersionInDB.Github.GitRef.Commit != "" {
		gitRef = modelPB.GitRef{
			Ref: &modelPB.GitRef_Commit{
				Commit: modelVersionInDB.Github.GitRef.Commit,
			},
		}
	}

	return &modelPB.ModelVersion{
		Version:     uint64(modelVersionInDB.Version),
		ModelId:     uint64(modelVersionInDB.ModelID),
		Description: modelVersionInDB.Description,
		CreatedAt:   timestamppb.New(modelVersionInDB.CreatedAt),
		UpdatedAt:   timestamppb.New(modelVersionInDB.UpdatedAt),
		Status:      st,
		Github: &modelPB.GitHub{
			RepoUrl: modelVersionInDB.Github.RepoUrl,
			GitRef:  &gitRef,
		},
	}
}

func createModelInfo(modelInDB datamodel.Model, versions []datamodel.Version, tritonModels []datamodel.TritonModel) *modelPB.Model {
	var vers []*modelPB.ModelVersion
	for i := 0; i < len(versions); i++ {
		vers = append(vers, createModelVersion(versions[i]))
	}
	return &modelPB.Model{
		Name:          modelInDB.Name,
		FullName:      modelInDB.FullName,
		Id:            uint64(modelInDB.ID),
		Task:          modelPB.Model_Task(modelInDB.Task),
		ModelVersions: vers,
	}
}

func setModelOnline(s *service, modelID uint, modelVersion uint) error {
	var tEnsembleModel datamodel.TritonModel
	var err error

	if tEnsembleModel, err = s.repository.GetTEnsembleModel(modelID, modelVersion); err != nil {
		return err
	}
	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err = s.repository.UpdateModelVersion(modelID, tEnsembleModel.ModelVersion, datamodel.Version{
			Status: datamodel.ValidStatus(modelPB.ModelVersion_STATUS_ERROR.String()),
		}); err != nil {
			return err
		}
		return err
	}

	if err = s.repository.UpdateModelVersion(modelID, tEnsembleModel.ModelVersion, datamodel.Version{
		Status: datamodel.ValidStatus(modelPB.ModelVersion_STATUS_ONLINE.String()),
	}); err != nil {
		return err
	}

	return nil
}

func setModelOffline(s *service, modelID uint, modelVersion uint) error {

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = s.repository.GetTModels(modelID); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err = s.repository.UpdateModelVersion(modelID, modelVersion, datamodel.Version{
				Status: datamodel.ValidStatus(modelPB.ModelVersion_STATUS_ERROR.String()),
			}); err != nil {
				return err
			}
			return err
		}
	}

	if err := s.repository.UpdateModelVersion(modelID, modelVersion, datamodel.Version{
		Status: datamodel.ValidStatus(modelPB.ModelVersion_STATUS_OFFLINE.String()),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) CreateModel(model *datamodel.Model) (*datamodel.Model, error) {
	// Validate the naming rule of model
	if match, _ := regexp.MatchString(util.MODEL_NAME_REGEX, model.Name); !match {
		return &datamodel.Model{}, status.Error(codes.FailedPrecondition, "The name of model is invalid")
	}

	if existingModel, _ := s.GetModelByName(model.Namespace, model.Name); existingModel.Name != "" {
		return &datamodel.Model{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", model.Name)
	}

	if err := s.repository.CreateModel(*model); err != nil {
		return &datamodel.Model{}, err
	}

	if createdModel, err := s.GetModelByName(model.Namespace, model.Name); err != nil {
		return &datamodel.Model{}, err
	} else {
		return &createdModel, nil
	}
}

func (s *service) GetModelByName(namespace string, modelName string) (datamodel.Model, error) {
	return s.repository.GetModelByName(namespace, modelName)
}

func (s *service) GetModelVersionLatest(modelId uint) (datamodel.Version, error) {
	return s.repository.GetModelVersionLatest(modelId)
}

func (s *service) CreateVersion(version datamodel.Version) (datamodel.Version, error) {
	if err := s.repository.CreateVersion(version); err != nil {
		return datamodel.Version{}, err
	}

	if createdVersion, err := s.repository.GetModelVersion(version.ModelID, version.Version); err != nil {
		return datamodel.Version{}, err
	} else {
		return createdVersion, nil
	}
}

func (s *service) GetModelVersion(modelId uint, version uint) (datamodel.Version, error) {
	return s.repository.GetModelVersion(modelId, version)
}

func (s *service) GetModelVersions(modelId uint) ([]datamodel.Version, error) {
	return s.repository.GetModelVersions(modelId)
}

func (s *service) GetTModels(modelId uint) ([]datamodel.TritonModel, error) {
	return s.repository.GetTModels(modelId)
}

func (s *service) ModelInfer(namespace string, modelName string, version uint, imgsBytes [][]byte, task modelPB.Model_Task) (interface{}, error) {
	// Triton model name is change into
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return nil, fmt.Errorf("Model not found")
	}

	ensembleModel, err := s.repository.GetTEnsembleModel(modelInDB.ID, version)
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

func createModel(s *service, namespace string, uploadedModel *datamodel.Model) (datamodel.Model, []datamodel.Version, []datamodel.TritonModel, error) {

	modelInDB, err := s.GetModelByName(namespace, uploadedModel.Name)
	if err != nil {
		createdModel, err := s.CreateModel(uploadedModel)
		if err != nil {
			return datamodel.Model{}, []datamodel.Version{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create model in DB")
		}
		modelInDB = *createdModel
	}
	latestVersion, err := s.repository.GetModelVersionLatest(modelInDB.ID)
	if err == nil {
		uploadedModel.Versions[0].Version = latestVersion.Version + 1
	} else {
		uploadedModel.Versions[0].Version = 1
	}
	uploadedModel.Versions[0].ModelID = modelInDB.ID
	versionInDB, err := s.CreateVersion(uploadedModel.Versions[0])
	if err != nil {
		return datamodel.Model{}, []datamodel.Version{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create model version in DB")
	}
	for i := 0; i < len(uploadedModel.TritonModels); i++ {
		tritonModel := uploadedModel.TritonModels[i]
		tritonModel.ModelID = modelInDB.ID
		tritonModel.ModelVersion = versionInDB.Version
		err = s.repository.CreateTModel(tritonModel)
		if err != nil {
			return datamodel.Model{}, []datamodel.Version{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create triton model in DB")
		}
	}
	versions, err := s.GetModelVersions(modelInDB.ID)
	if err != nil {
		return datamodel.Model{}, []datamodel.Version{}, []datamodel.TritonModel{}, fmt.Errorf("Could not get model versions in DB")
	}

	return modelInDB, versions, uploadedModel.TritonModels, nil
}

func (s *service) CreateModelBinaryFileUpload(namespace string, uploadedModel *datamodel.Model) (*modelPB.Model, error) {
	modelInDB, versions, tritonModels, err := createModel(s, namespace, uploadedModel)
	return createModelInfo(modelInDB, versions, tritonModels), err
}

func (s *service) ListModels(namespace string) ([]*modelPB.Model, error) {
	models, err := s.repository.ListModels(datamodel.ListModelQuery{Namespace: namespace})
	if err != nil {
		return []*modelPB.Model{}, err
	}

	var resModels []*modelPB.Model
	for i := 0; i < len(models); i++ {
		md := models[i]
		versions, err := s.GetModelVersions(md.ID)
		if err != nil {
			return []*modelPB.Model{}, err
		}
		tritonModels, err := s.GetTModels(md.ID)
		if err != nil {
			return []*modelPB.Model{}, err

		}
		resModels = append(resModels, createModelInfo(md, versions, tritonModels))
	}

	return resModels, nil
}

func (s *service) UpdateModelVersion(namespace string, in *modelPB.UpdateModelVersionRequest) (*modelPB.ModelVersion, error) {
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
					if err := setModelOnline(s, modelInDB.ID, uint(in.Version)); err != nil {
						return &modelPB.ModelVersion{}, err
					}
				case modelPB.ModelVersion_STATUS_OFFLINE:
					if err := setModelOffline(s, modelInDB.ID, uint(in.Version)); err != nil {
						return &modelPB.ModelVersion{}, err
					}
				default:
					return &modelPB.ModelVersion{}, fmt.Errorf("Wrong status value. Status should be ONLINE or OFFLINE")
				}
			case "description":
				err = s.repository.UpdateModelVersion(modelInDB.ID, uint(in.Version), datamodel.Version{
					Description: in.VersionPatch.Description,
				})
				if err != nil {
					return &modelPB.ModelVersion{}, err
				}
			}
		}
	}
	modelVersionInDB, err := s.GetModelVersion(modelInDB.ID, uint(in.Version))
	return createModelVersion(modelVersionInDB), err
}

func (s *service) GetFullModelData(namespace string, modelName string) (*modelPB.Model, error) {
	// TODO: improve by using join
	resModelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return &modelPB.Model{}, err
	}

	versions, err := s.GetModelVersions(resModelInDB.ID)
	if err != nil {
		return &modelPB.Model{}, err
	}

	tritonModels, err := s.GetTModels(resModelInDB.ID)
	if err != nil {
		return &modelPB.Model{}, err
	}

	return createModelInfo(resModelInDB, versions, tritonModels), nil
}

func (s *service) DeleteModel(namespace string, modelName string) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelVersionsInDB, err := s.GetModelVersions(modelInDB.ID)
	if err == nil {
		for i := 0; i < len(modelVersionsInDB); i++ {
			if err := setModelOffline(s, modelInDB.ID, modelVersionsInDB[i].Version); err != nil {
				return err
			}
		}
		tritonModels, err := s.repository.GetTModels(modelInDB.ID)
		if err == nil {
			for i := 0; i < len(tritonModels); i++ {
				modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
				_ = os.RemoveAll(modelDir)
			}
		}
	}

	return s.repository.DeleteModel(modelInDB.ID)
}

func (s *service) DeleteModelVersion(namespace string, modelName string, version uint) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelVersionInDB, err := s.GetModelVersion(modelInDB.ID, version)
	if err != nil {
		return err
	}

	if err := setModelOffline(s, modelInDB.ID, modelVersionInDB.Version); err != nil {
		return err
	}

	tritonModels, err := s.repository.GetTModelVersions(modelInDB.ID, modelVersionInDB.Version)
	if err == nil {
		for i := 0; i < len(tritonModels); i++ {
			modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
			_ = os.RemoveAll(modelDir)
		}
	}

	modelVersionsInDB, err := s.GetModelVersions(modelInDB.ID)
	if err != nil {
		return err
	}

	if len(modelVersionsInDB) > 1 {
		return s.repository.DeleteModelVersion(modelInDB.ID, modelVersionInDB.Version)
	} else {
		return s.repository.DeleteModel(modelInDB.ID)
	}
}
