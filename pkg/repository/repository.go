package repository

import (
	"fmt"

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
	CreateVersion(version datamodel.Version) error
	UpdateModelVersion(modelId uint, modelVersion uint, versionInfo datamodel.Version) error
	GetModelVersion(modelId uint, version uint) (datamodel.Version, error)
	GetModelVersions(modelId uint) ([]datamodel.Version, error)
	UpdateModelMetaData(modelId uint, updatedModel datamodel.Model) error
	CreateTModel(model datamodel.TritonModel) error
	GetTritonModels(modelId uint) ([]datamodel.TritonModel, error)
	GetTritonEnsembleModel(modelId uint, version uint) (datamodel.TritonModel, error)
	DeleteModel(modelId uint) error
	DeleteModelVersion(modelId uint, version uint) error
	GetModelVersionLatest(modelId uint) (datamodel.Version, error)
	GetTritonModelVersions(modelId uint, version uint) ([]datamodel.TritonModel, error)
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
	`"model"."task"`,
	`"model"."visibility"`,
	`"model"."source"`,
	`CONCAT(namespace, '/', name) as full_name`,
}

func (r *repository) CreateModel(model datamodel.Model) error {
	l, _ := logger.GetZapLogger()
	// We ignore the full_name column since it's a virtual column
	if result := r.db.Model(&datamodel.Model{}).Omit("TritonModels", "Versions", "FullName").Create(&model); result.Error != nil {
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

func (r *repository) CreateVersion(version datamodel.Version) error {
	if result := r.db.Model(&datamodel.Version{}).Create(&version); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) UpdateModelVersion(modelId uint, modelVersion uint, versionInfo datamodel.Version) error {

	if result := r.db.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId, "version": modelVersion}).Updates(&versionInfo); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetModelVersion(modelId uint, version uint) (datamodel.Version, error) {
	var versionDB datamodel.Version
	if result := r.db.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId, "version": version}).First(&versionDB); result.Error != nil {
		return datamodel.Version{}, status.Errorf(codes.NotFound, "The version %v for model %v not found", version, modelId)
	}
	return versionDB, nil
}

func (r *repository) GetModelVersions(modelId uint) ([]datamodel.Version, error) {
	var versions []datamodel.Version
	if result := r.db.Model(&datamodel.Version{}).Where("model_id", modelId).Order("version asc").Find(&versions); result.Error != nil {
		return []datamodel.Version{}, status.Errorf(codes.NotFound, "The versions for model %v not found", modelId)
	}
	return versions, nil
}

func (r *repository) UpdateModelMetaData(modelId uint, updatedModel datamodel.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.db.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where("id", modelId).Omit("TritonModels", "Versions", "FullName").Updates(&updatedModel); result.Error != nil {
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

func (r *repository) GetTritonModels(modelId uint) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where("model_id", modelId).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *repository) GetTritonEnsembleModel(modelId uint, version uint) (datamodel.TritonModel, error) {
	var ensembleModel datamodel.TritonModel
	result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_id": modelId, "model_version": version, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelId)
	}
	return ensembleModel, nil
}

func (r *repository) GetTritonModelVersions(modelId uint, version uint) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_id": modelId, "model_version": version}).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelId)
	}
	return tmodels, nil
}

func (r *repository) DeleteModel(modelId uint) error {
	if result := r.db.Model(&datamodel.Model{}).Select("Versions", "TritonModels").Delete(&datamodel.Model{Model: gorm.Model{ID: modelId}}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelId)
	}
	return nil
}

func (r *repository) GetModelVersionLatest(modelId uint) (datamodel.Version, error) {
	var versionDB datamodel.Version
	if result := r.db.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId}).Order("version desc").First(&versionDB); result.Error != nil {
		return datamodel.Version{}, status.Errorf(codes.NotFound, "There is no version for model id %v not found", modelId)
	}
	return versionDB, nil
}

func (r *repository) DeleteModelVersion(modelId uint, version uint) error {
	if result := r.db.Model(&datamodel.Version{}).Where(map[string]interface{}{"model_id": modelId, "version": version}).Select("TritonModels").Delete(&datamodel.Version{}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v and version %v", modelId, version)
	}
	return nil
}
