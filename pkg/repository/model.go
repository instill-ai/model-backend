package repository

import (
	"fmt"

	"github.com/gogo/status"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/pkg/models"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type ModelRepository interface {
	CreateModel(model models.Model) error
	GetModelByName(namespace string, modelName string) (models.Model, error)
	ListModels(query models.ListModelQuery) ([]models.Model, error)
	CreateVersion(version models.Version) error
	UpdateModelVersions(modelId int32, version models.Version) error
	GetModelVersion(modelId int32, version int32) (models.Version, error)
	GetModelVersions(modelId int32) ([]models.Version, error)
	UpdateModelMetaData(modelId int32, updatedModel models.Model) error
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
	`"models"."id" as id`,
	`"models"."name"`,
	`"models"."optimized"`,
	`"models"."description"`,
	`"models"."type"`,
	`"models"."framework"`,
	`"models"."created_at"`,
	`"models"."updated_at"`,
	`"models"."organization"`,
	`"models"."icon"`,
	`"models"."visibility"`,
	`"models"."author"`,
	`'models' as kind`,
	`CONCAT(namespace, '/', name) as full_name`,
}

func (r *modelRepository) CreateModel(model models.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&models.Model{}).Omit(`"models"."full_name"`).Create(&model); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) GetModelByName(namespace string, modelName string) (models.Model, error) {
	var model models.Model
	if result := r.DB.Model(&models.Model{}).Select(GetModelSelectedFields).Where(map[string]interface{}{"name": modelName, "namespace": namespace}).First(&model); result.Error != nil {
		return models.Model{}, status.Errorf(codes.NotFound, "The model name %s you specified is not found in namespace %s", modelName, namespace)
	}

	return model, nil
}

func (r *modelRepository) ListModels(query models.ListModelQuery) ([]models.Model, error) {
	var modelList []models.Model
	r.DB.Model(&models.Model{}).Select(GetModelSelectedFields).Where("namespace", query.Namespace).Find(&modelList)

	return modelList, nil
}

func (r *modelRepository) CreateVersion(version models.Version) error {

	if result := r.DB.Model(&models.Version{}).Omit(`"version"."model_name"`).Create(&version); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) UpdateModelVersions(modelId int32, version models.Version) error {

	if result := r.DB.Model(&models.Version{}).Omit(`"version"."model_name"`).Where("model_id", modelId).Updates(&version); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *modelRepository) GetModelVersion(modelId int32, version int32) (models.Version, error) {
	var versionDB models.Version
	if result := r.DB.Model(&models.Version{}).Omit(`"versions"."model_name"`).Where(map[string]interface{}{"model_id": modelId, "version": version}).First(&versionDB); result.Error != nil {
		return models.Version{}, status.Errorf(codes.NotFound, "The version %v for model %v not found", version, modelId)
	}

	return versionDB, nil
}

func (r *modelRepository) GetModelVersions(modelId int32) ([]models.Version, error) {
	var versions []models.Version
	if result := r.DB.Model(&models.Version{}).Omit(`"versions"."model_name"`).Where("model_id", modelId).Find(&versions); result.Error != nil {
		return []models.Version{}, status.Errorf(codes.NotFound, "The versions for model %v not found", modelId)
	}
	return versions, nil
}

func (r *modelRepository) UpdateModelMetaData(modelId int32, updatedModel models.Model) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&models.Model{}).Select(GetModelSelectedFields).Where("id", modelId).Updates(&updatedModel); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}
