package repository

import (
	"fmt"

	"github.com/gogo/status"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type ModelRepository interface {
	CreateModel(model datamodel.Model) error
	GetModelByName(namespace string, modelName string) (datamodel.Model, error)
	ListModels(query datamodel.ListModelQuery) ([]datamodel.Model, error)
	CreateVersion(version datamodel.Version) error
	UpdateModelVersion(modelId uint64, modelVersion uint64, versionInfo datamodel.Version) error
	GetModelVersion(modelId uint64, version uint64) (datamodel.Version, error)
	GetModelVersions(modelId uint64) ([]datamodel.Version, error)
	UpdateModelMetaData(modelId uint64, updatedModel datamodel.Model) error
	CreateTModel(model datamodel.TModel) error
	GetTModels(modelId uint64) ([]datamodel.TModel, error)
	GetTEnsembleModel(modelId uint64, version uint64) (datamodel.TModel, error)
	DeleteModel(modelId uint64) error
	DeleteModelVersion(modelId uint64, version uint64) error
	GetModelVersionLatest(modelId uint64) (datamodel.Version, error)
	GetTModelVersions(modelId uint64, version uint64) ([]datamodel.TModel, error)
}

type modelRepository struct {
	DB *gorm.DB
}

func NewModelRepository(db *gorm.DB) ModelRepository {
	return &modelRepository{
		DB: db,
	}
}

var GetModelSelectedFields = []string{
	`"models"."id"`,
	`"models"."name"`,
	`"models"."task"`,
	`CONCAT(namespace, '/', name) as full_name`,
}

func (r *modelRepository) CreateModel(model datamodel.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&datamodel.Model{}).Omit("TritonModels", "Versions", "FullName").Create(&model); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) GetModelByName(namespace string, modelName string) (datamodel.Model, error) {
	var model datamodel.Model
	if result := r.DB.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where(&datamodel.Model{Name: modelName, Namespace: namespace}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model name %s you specified is not found in namespace %s", modelName, namespace)
	}
	return model, nil
}

func (r *modelRepository) ListModels(query datamodel.ListModelQuery) ([]datamodel.Model, error) {
	var modelList []datamodel.Model
	r.DB.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where("namespace", query.Namespace).Find(&modelList)

	return modelList, nil
}

func (r *modelRepository) CreateVersion(version datamodel.Version) error {
	if result := r.DB.Model(&datamodel.Version{}).Omit(`"version"."model_name"`).Create(&version); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) UpdateModelVersion(modelId uint64, modelVersion uint64, versionInfo datamodel.Version) error {

	if result := r.DB.Model(&datamodel.Version{}).Omit(`"version"."model_name"`).Where(map[string]interface{}{"model_id": modelId, "version": modelVersion}).Updates(&versionInfo); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) GetModelVersion(modelId uint64, version uint64) (datamodel.Version, error) {
	var versionDB datamodel.Version
	if result := r.DB.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId, "version": version}).First(&versionDB); result.Error != nil {
		return datamodel.Version{}, status.Errorf(codes.NotFound, "The version %v for model %v not found", version, modelId)
	}
	return versionDB, nil
}

func (r *modelRepository) GetModelVersions(modelId uint64) ([]datamodel.Version, error) {
	var versions []datamodel.Version
	if result := r.DB.Model(&datamodel.Version{}).Where("model_id", modelId).Order("version asc").Find(&versions); result.Error != nil {
		return []datamodel.Version{}, status.Errorf(codes.NotFound, "The versions for model %v not found", modelId)
	}
	return versions, nil
}

func (r *modelRepository) UpdateModelMetaData(modelId uint64, updatedModel datamodel.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where("id", modelId).Omit("TritonModels", "Versions", "FullName").Updates(&updatedModel); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) CreateTModel(model datamodel.TModel) error {
	if result := r.DB.Model(&datamodel.TModel{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) GetTModels(modelId uint64) ([]datamodel.TModel, error) {
	var tmodels []datamodel.TModel
	if result := r.DB.Model(&datamodel.TModel{}).Where("model_id", modelId).Find(&tmodels); result.Error != nil {
		return []datamodel.TModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *modelRepository) GetTEnsembleModel(modelId uint64, version uint64) (datamodel.TModel, error) {
	var ensembleModel datamodel.TModel
	result := r.DB.Model(&datamodel.TModel{}).Where(map[string]interface{}{"model_id": modelId, "model_version": version, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return datamodel.TModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelId)
	}
	return ensembleModel, nil
}

func (r *modelRepository) GetTModelVersions(modelId uint64, version uint64) ([]datamodel.TModel, error) {
	var tmodels []datamodel.TModel
	if result := r.DB.Model(&datamodel.TModel{}).Where(map[string]interface{}{"model_id": modelId, "model_version": version}).Find(&tmodels); result.Error != nil {
		return []datamodel.TModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *modelRepository) DeleteModel(modelId uint64) error {
	if result := r.DB.Model(&datamodel.Model{}).Where("id", modelId).Delete(datamodel.Model{}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelId)
	}
	return nil
}

func (r *modelRepository) GetModelVersionLatest(modelId uint64) (datamodel.Version, error) {
	var versionDB datamodel.Version
	if result := r.DB.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId}).Order("version desc").First(&versionDB); result.Error != nil {
		return datamodel.Version{}, status.Errorf(codes.NotFound, "There is no version for model id %v not found", modelId)
	}
	return versionDB, nil
}

func (r *modelRepository) DeleteModelVersion(modelId uint64, version uint64) error {
	if result := r.DB.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId, "version": version}).Delete(datamodel.Version{}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v and version %v", modelId, version)
	}
	return nil
}
