package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
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
	CreateInstance(instance datamodel.Instance) (datamodel.Instance, error)
	GetModelInstance(modelId uuid.UUID, instanceName string) (datamodel.Instance, error)
	GetModelInstances(modelId uuid.UUID) ([]datamodel.Instance, error)
	GetModelInstanceLatest(modelId uuid.UUID) (datamodel.Instance, error)
	GetFullModelData(namespace string, modelName string) (*modelPB.ModelDefinition, error)
	ModelInfer(namespace string, modelName string, instanceName string, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error)
	CreateModelBinaryFileUpload(namespace string, createdModel *datamodel.Model) (*modelPB.ModelDefinition, error)
	ListModels(namespace string) ([]*modelPB.ModelDefinition, error)
	UpdateModelInstance(namespace string, updatedInfo *modelPB.UpdateModelInstanceRequest) (*modelPB.ModelInstance, error)
	DeleteModel(namespace string, modelName string) error
	DeleteModelInstance(namespace string, modelName string, instanceName string) error
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

func createModelInstance(modelInDB datamodel.Model, modelInstanceInDB datamodel.Instance) *modelPB.ModelInstance {
	var st = modelPB.ModelInstance_STATUS_OFFLINE
	if string(modelInstanceInDB.Status) == modelPB.ModelInstance_STATUS_ONLINE.String() {
		st = modelPB.ModelInstance_STATUS_ONLINE
	} else if string(modelInstanceInDB.Status) == modelPB.ModelInstance_STATUS_ERROR.String() {
		st = modelPB.ModelInstance_STATUS_ERROR
	}

	var configuration modelPB.Configuration
	var githubConfigObj datamodel.InstanceConfiguration
	_ = json.Unmarshal(modelInstanceInDB.Config, &githubConfigObj)

	var source = modelPB.ModelDefinition_SOURCE_GITHUB
	if modelInDB.Source == datamodel.ModelDefinitionSource(modelPB.ModelDefinition_SOURCE_GITHUB) {
		configuration = modelPB.Configuration{
			Repo:    githubConfigObj.Repo,
			Tag:     githubConfigObj.Tag,
			HtmlUrl: githubConfigObj.HtmlUrl,
		}
	} else if modelInDB.Source == datamodel.ModelDefinitionSource(modelPB.ModelDefinition_SOURCE_LOCAL) {
		configuration = modelPB.Configuration{}
		source = modelPB.ModelDefinition_SOURCE_LOCAL
	}

	return &modelPB.ModelInstance{
		Id:                    modelInstanceInDB.ID.String(),
		Name:                  modelInstanceInDB.Name,
		ModelDefinitionName:   modelInDB.Name,
		CreatedAt:             timestamppb.New(modelInstanceInDB.CreatedAt),
		UpdatedAt:             timestamppb.New(modelInstanceInDB.UpdatedAt),
		Status:                st,
		Configuration:         &configuration,
		Task:                  modelPB.ModelInstance_Task(modelInstanceInDB.Task),
		ModelDefinitionSource: source,
		ModelDefinitionId:     modelInDB.ID.String(),
	}
}

func createModelInfo(modelInDB datamodel.Model, modelInstances []datamodel.Instance, tritonModels []datamodel.TritonModel) *modelPB.ModelDefinition {
	var instances []*modelPB.ModelInstance
	for i := 0; i < len(modelInstances); i++ {
		instances = append(instances, createModelInstance(modelInDB, modelInstances[i]))
	}
	visibility := modelPB.ModelDefinition_VISIBILITY_PUBLIC
	if modelInDB.Visibility == datamodel.ModelDefinitionVisibility(modelPB.ModelDefinition_VISIBILITY_PRIVATE) {
		visibility = modelPB.ModelDefinition_VISIBILITY_PRIVATE
	}

	var source = modelPB.ModelDefinition_SOURCE_LOCAL
	var config modelPB.Configuration
	if modelInDB.Source == datamodel.ModelDefinitionSource(modelPB.ModelDefinition_SOURCE_GITHUB) {
		source = modelPB.ModelDefinition_SOURCE_GITHUB
		_ = json.Unmarshal(modelInDB.Config, &config)
	}

	var owner modelPB.Owner
	_ = json.Unmarshal(modelInDB.Owner, &owner)
	return &modelPB.ModelDefinition{
		Id:            modelInDB.ID.String(),
		Name:          modelInDB.Name,
		FullName:      modelInDB.FullName,
		Visibility:    visibility,
		Instances:     instances,
		Source:        source,
		Configuration: &config,
		Owner:         &owner,
		CreatedAt:     timestamppb.New(modelInDB.CreatedAt),
		UpdatedAt:     timestamppb.New(modelInDB.UpdatedAt),
		Description:   modelInDB.Description,
	}
}

func setModelOnline(s *service, modelId uuid.UUID, instanceName string) error {
	var tEnsembleModel datamodel.TritonModel
	var err error

	if tEnsembleModel, err = s.repository.GetTritonEnsembleModel(modelId, instanceName); err != nil {
		return err
	}
	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err = s.repository.UpdateModelInstance(modelId, tEnsembleModel.ModelInstance, datamodel.Instance{
			Status: datamodel.ModelInstanceStatus(modelPB.ModelInstance_STATUS_ERROR),
		}); err != nil {
			return err
		}
		return err
	}

	if err = s.repository.UpdateModelInstance(modelId, tEnsembleModel.ModelInstance, datamodel.Instance{
		Status: datamodel.ModelInstanceStatus(modelPB.ModelInstance_STATUS_ONLINE),
	}); err != nil {
		return err
	}

	return nil
}

func setModelOffline(s *service, modelId uuid.UUID, instanceName string) error {

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = s.repository.GetTritonModels(modelId); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err = s.repository.UpdateModelInstance(modelId, instanceName, datamodel.Instance{
				Status: datamodel.ModelInstanceStatus(modelPB.ModelInstance_STATUS_ERROR),
			}); err != nil {
				return err
			}
			return err
		}
	}

	if err := s.repository.UpdateModelInstance(modelId, instanceName, datamodel.Instance{
		Status: datamodel.ModelInstanceStatus(modelPB.ModelInstance_STATUS_OFFLINE),
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

func (s *service) GetModelInstanceLatest(modelId uuid.UUID) (datamodel.Instance, error) {
	return s.repository.GetModelInstanceLatest(modelId)
}

func (s *service) CreateInstance(instance datamodel.Instance) (datamodel.Instance, error) {
	if err := s.repository.CreateInstance(instance); err != nil {
		return datamodel.Instance{}, err
	}

	if createdInstance, err := s.repository.GetModelInstance(instance.ModelID, instance.Name); err != nil {
		return datamodel.Instance{}, err
	} else {
		return createdInstance, nil
	}
}

func (s *service) GetModelInstance(modelId uuid.UUID, instanceName string) (datamodel.Instance, error) {
	return s.repository.GetModelInstance(modelId, instanceName)
}

func (s *service) GetModelInstances(modelId uuid.UUID) ([]datamodel.Instance, error) {
	return s.repository.GetModelInstances(modelId)
}

func (s *service) GetTModels(modelId uuid.UUID) ([]datamodel.TritonModel, error) {
	return s.repository.GetTritonModels(modelId)
}

func (s *service) ModelInfer(namespace string, modelName string, instanceName string, imgsBytes [][]byte, task modelPB.ModelInstance_Task) (interface{}, error) {
	// Triton model name is change into
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return nil, fmt.Errorf("Model not found")
	}

	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelInDB.BaseDynamic.ID, instanceName)
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

func createModel(s *service, namespace string, uploadedModel *datamodel.Model) (datamodel.Model, []datamodel.Instance, []datamodel.TritonModel, error) {

	modelInDB, err := s.GetModelByName(namespace, uploadedModel.Name)
	if err != nil {
		createdModel, err := s.CreateModel(uploadedModel)
		if err != nil {
			return datamodel.Model{}, []datamodel.Instance{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create model in DB")
		}
		modelInDB = *createdModel
	}

	uploadedModel.Instances[0].ModelID = modelInDB.ID
	instanceInDB, err := s.CreateInstance(uploadedModel.Instances[0])
	if err != nil {
		return datamodel.Model{}, []datamodel.Instance{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create model instance in DB")
	}
	for i := 0; i < len(uploadedModel.TritonModels); i++ {
		tritonModel := uploadedModel.TritonModels[i]
		tritonModel.ModelID = modelInDB.ID
		tritonModel.ModelInstance = instanceInDB.Name
		err = s.repository.CreateTModel(tritonModel)
		if err != nil {
			return datamodel.Model{}, []datamodel.Instance{}, []datamodel.TritonModel{}, fmt.Errorf("Could not create triton model in DB")
		}
	}
	instances, err := s.GetModelInstances(modelInDB.ID)
	if err != nil {
		return datamodel.Model{}, []datamodel.Instance{}, []datamodel.TritonModel{}, fmt.Errorf("Could not get model instances in DB")
	}

	return modelInDB, instances, uploadedModel.TritonModels, nil
}

func (s *service) CreateModelBinaryFileUpload(namespace string, uploadedModel *datamodel.Model) (*modelPB.ModelDefinition, error) {
	modelInDB, instances, tritonModels, err := createModel(s, namespace, uploadedModel)
	return createModelInfo(modelInDB, instances, tritonModels), err
}

func (s *service) ListModels(namespace string) ([]*modelPB.ModelDefinition, error) {
	models, err := s.repository.ListModels(datamodel.ListModelQuery{Namespace: namespace})
	if err != nil {
		return []*modelPB.ModelDefinition{}, err
	}
	var resModels []*modelPB.ModelDefinition
	for i := 0; i < len(models); i++ {
		md := models[i]
		instances, err := s.GetModelInstances(md.ID)
		if err != nil {
			return []*modelPB.ModelDefinition{}, err
		}
		tritonModels, err := s.GetTModels(md.ID)
		if err != nil {
			return []*modelPB.ModelDefinition{}, err

		}
		resModels = append(resModels, createModelInfo(md, instances, tritonModels))
	}

	return resModels, nil
}
func (s *service) UpdateModelInstance(namespace string, in *modelPB.UpdateModelInstanceRequest) (*modelPB.ModelInstance, error) {
	modelInDB, err := s.GetModelByName(namespace, in.ModelName)
	if err != nil {
		return &modelPB.ModelInstance{}, err
	}

	if _, err = s.GetModelInstance(modelInDB.ID, in.InstanceName); err != nil {
		return &modelPB.ModelInstance{}, err
	}

	switch in.Status {
	case modelPB.ModelInstance_STATUS_ONLINE:
		if err := setModelOnline(s, modelInDB.ID, in.InstanceName); err != nil {
			return &modelPB.ModelInstance{}, err
		}
	case modelPB.ModelInstance_STATUS_OFFLINE:
		if err := setModelOffline(s, modelInDB.ID, in.InstanceName); err != nil {
			return &modelPB.ModelInstance{}, err
		}
	default:
		return &modelPB.ModelInstance{}, fmt.Errorf("Wrong status value. Status should be ONLINE or OFFLINE")
	}

	modelInstanceInDB, err := s.GetModelInstance(modelInDB.ID, in.InstanceName)
	if err != nil {
		return &modelPB.ModelInstance{}, err
	}

	return createModelInstance(modelInDB, modelInstanceInDB), err
}

func (s *service) GetFullModelData(namespace string, modelName string) (*modelPB.ModelDefinition, error) {
	// TODO: improve by using join
	resModelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return &modelPB.ModelDefinition{}, err
	}

	instances, err := s.GetModelInstances(resModelInDB.ID)
	if err != nil {
		return &modelPB.ModelDefinition{}, err
	}

	tritonModels, err := s.GetTModels(resModelInDB.ID)
	if err != nil {
		return &modelPB.ModelDefinition{}, err
	}

	return createModelInfo(resModelInDB, instances, tritonModels), nil
}

func (s *service) DeleteModel(namespace string, modelName string) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelInstanceInDB, err := s.GetModelInstances(modelInDB.ID)
	if err == nil {
		for i := 0; i < len(modelInstanceInDB); i++ {
			if err := setModelOffline(s, modelInDB.ID, modelInstanceInDB[i].Name); err != nil {
				return err
			}
			// remove README.md
			_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md#%v", configs.Config.TritonServer.ModelStore, namespace, modelName, modelInstanceInDB[i].Name))
		}
		tritonModels, err := s.repository.GetTritonModels(modelInDB.ID)
		if err == nil {
			// remove model folders
			for i := 0; i < len(tritonModels); i++ {
				modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
				_ = os.RemoveAll(modelDir)
			}
		}
	}

	return s.repository.DeleteModel(modelInDB.ID)
}

func (s *service) DeleteModelInstance(namespace string, modelName string, instanceName string) error {
	modelInDB, err := s.GetModelByName(namespace, modelName)
	if err != nil {
		return err
	}
	modelInstanceInDB, err := s.GetModelInstance(modelInDB.ID, instanceName)
	if err != nil {
		return err
	}

	if err := setModelOffline(s, modelInDB.ID, modelInstanceInDB.Name); err != nil {
		return err
	}

	tritonModels, err := s.repository.GetTritonModelVersions(modelInDB.ID, modelInstanceInDB.Name)
	if err == nil {
		for i := 0; i < len(tritonModels); i++ {
			modelDir := filepath.Join(configs.Config.TritonServer.ModelStore, tritonModels[i].Name)
			_ = os.RemoveAll(modelDir)
		}
	}
	// remove README.md
	_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md#%v", configs.Config.TritonServer.ModelStore, namespace, modelName, instanceName))

	modelInstancesInDB, err := s.GetModelInstances(modelInDB.ID)
	if err != nil {
		return err
	}

	if len(modelInstancesInDB) > 1 {
		return s.repository.DeleteModelInstance(modelInDB.ID, modelInstanceInDB.Name)
	} else {
		return s.repository.DeleteModel(modelInDB.ID)
	}
}
