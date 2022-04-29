package repository

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

type Repository interface {
	CreateModel(model datamodel.Model) error
	GetModelByName(namespace string, modelName string) (datamodel.Model, error)
	ListModels(query datamodel.ListModelQuery) ([]datamodel.Model, error)
	CreateInstance(instance datamodel.Instance) error
	UpdateModelInstance(modelId uuid.UUID, modelInstance string, instanceInfo datamodel.Instance) error
	GetModelInstance(modelId uuid.UUID, instanceName string) (datamodel.Instance, error)
	GetModelInstances(modelId uuid.UUID) ([]datamodel.Instance, error)
	UpdateModelMetaData(modelId uuid.UUID, updatedModel datamodel.Model) error
	CreateTModel(model datamodel.TritonModel) error
	GetTritonModels(modelId uuid.UUID) ([]datamodel.TritonModel, error)
	GetTritonEnsembleModel(modelId uuid.UUID, instanceName string) (datamodel.TritonModel, error)
	DeleteModel(modelId uuid.UUID) error
	DeleteModelInstance(modelId uuid.UUID, instanceName string) error
	GetModelInstanceLatest(modelId uuid.UUID) (datamodel.Instance, error)
	GetTritonModelVersions(modelId uuid.UUID, instanceName string) ([]datamodel.TritonModel, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

var GetModelSelectedFields = []string{
	`"model"."id"`,
	`"model"."name"`,
	`"model"."visibility"`,
	`"model"."source"`,
	`"model"."owner"`,
	`"model"."config"`,
	`"model"."description"`,
	`"model"."created_at"`,
	`"model"."updated_at"`,
	`CONCAT(namespace, '/', name) as full_name`,
}

func (r *repository) CreateModel(model datamodel.Model) error {
	l, _ := logger.GetZapLogger()
	// We ignore the full_name column since it's a virtual column
	if result := r.db.Model(&datamodel.Model{}).Omit("TritonModels", "Instances", "FullName").Create(&model); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetModelByName(namespace string, modelName string) (datamodel.Model, error) {
	var model datamodel.Model
	if result := r.db.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where(&datamodel.Model{Name: modelName, Namespace: namespace}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model name %s you specified is not found in namespace %s", modelName, namespace)
	}
	return model, nil
}

func (r *repository) ListModels(query datamodel.ListModelQuery) ([]datamodel.Model, error) {
	var modelList []datamodel.Model
	r.db.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where("namespace", query.Namespace).Find(&modelList)
	return modelList, nil
}

func (r *repository) CreateInstance(instance datamodel.Instance) error {
	if result := r.db.Model(&datamodel.Instance{}).Omit("Source", "ModelDefinitionId").Create(&instance); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) UpdateModelInstance(modelId uuid.UUID, instanceName string, instanceInfo datamodel.Instance) error {

	if result := r.db.Model(&datamodel.Instance{}).Omit("Source", "ModelDefinitionId").Where(map[string]interface{}{"model_id": modelId, "name": instanceName}).Updates(&instanceInfo); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetModelInstance(modelId uuid.UUID, instanceName string) (datamodel.Instance, error) {
	var instanceDB datamodel.Instance
	if result := r.db.Model(&datamodel.Instance{}).Omit("Source", "ModelDefinitionId").Where(map[string]interface{}{"model_id": modelId, "name": instanceName}).First(&instanceDB); result.Error != nil {
		return datamodel.Instance{}, status.Errorf(codes.NotFound, "The instance %v for model %v not found", instanceName, modelId)
	}
	return instanceDB, nil
}

func (r *repository) GetModelInstances(modelId uuid.UUID) ([]datamodel.Instance, error) {
	var instances []datamodel.Instance
	if result := r.db.Model(&datamodel.Instance{}).Omit("Source", "ModelDefinitionId").Where("model_id", modelId).Order("name asc").Find(&instances); result.Error != nil {
		return []datamodel.Instance{}, status.Errorf(codes.NotFound, "The instance for model %v not found", modelId)
	}
	return instances, nil
}

func (r *repository) UpdateModelMetaData(modelId uuid.UUID, updatedModel datamodel.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.db.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where("id", modelId).Omit("TritonModels", "Instances", "FullName").Updates(&updatedModel); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) CreateTModel(model datamodel.TritonModel) error {
	if result := r.db.Model(&datamodel.TritonModel{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetTritonModels(modelId uuid.UUID) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where("model_id", modelId).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *repository) GetTritonEnsembleModel(modelId uuid.UUID, instanceName string) (datamodel.TritonModel, error) {
	var ensembleModel datamodel.TritonModel
	result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_id": modelId, "model_instance": instanceName, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelId)
	}
	return ensembleModel, nil
}

func (r *repository) GetTritonModelVersions(modelId uuid.UUID, instanceName string) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_id": modelId, "model_instance": instanceName}).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *repository) DeleteModel(modelId uuid.UUID) error {
	if result := r.db.Model(&datamodel.Model{}).Select("Instances", "TritonModels").Delete(&datamodel.Model{BaseDynamic: datamodel.BaseDynamic{ID: modelId}}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelId)
	}
	return nil
}

func (r *repository) GetModelInstanceLatest(modelId uuid.UUID) (datamodel.Instance, error) {
	var instanceDB datamodel.Instance
	if result := r.db.Model(&datamodel.Instance{}).Where(map[string]interface{}{"model_id": modelId}).Order("instance desc").First(&instanceDB); result.Error != nil {
		return datamodel.Instance{}, status.Errorf(codes.NotFound, "There is no instance for model id %v not found", modelId)
	}
	return instanceDB, nil
}

func (r *repository) DeleteModelInstance(modelId uuid.UUID, instanceName string) error {
	if result := r.db.Model(&datamodel.Instance{}).Where(map[string]interface{}{"model_id": modelId, "name": instanceName}).Select("TritonModels").Delete(&datamodel.Instance{}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v and instance name %v", modelId, instanceName)
	}
	return nil
}
