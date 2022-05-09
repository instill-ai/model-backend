package repository

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/internal/paginate"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

type Repository interface {
	CreateModel(model datamodel.Model) error
	GetModelById(owner string, modelId string) (datamodel.Model, error)
	DeleteModel(modelUid uuid.UUID) error
	UpdateModel(modelUid uuid.UUID, updatedModel datamodel.Model) error
	ListModel(owner string, view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error)
	CreateModelInstance(instance datamodel.ModelInstance) error
	UpdateModelInstance(modelInstanceUID uuid.UUID, instanceInfo datamodel.ModelInstance) error
	GetModelInstance(modelUid uuid.UUID, instanceId string) (datamodel.ModelInstance, error)
	GetModelInstances(modelUid uuid.UUID) ([]datamodel.ModelInstance, error)
	ListModelInstance(modelUid uuid.UUID, view modelPB.View, pageSize int, pageToken string) (instances []datamodel.ModelInstance, nextPageToken string, totalSize int64, err error)
	CreateTritonModel(model datamodel.TritonModel) error
	GetTritonModels(modelInstanceUID uuid.UUID) ([]datamodel.TritonModel, error)
	GetTritonEnsembleModel(modelInstanceUID uuid.UUID) (datamodel.TritonModel, error)
	GetModelDefinition(id string) (datamodel.ModelDefinition, error)
	ListModelDefinition(view modelPB.View, pageSize int, pageToken string) (definitions []datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error)
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
	`CONCAT('models/', id) as name`,
	`"model"."uid"`,
	`"model"."id"`,
	`"model"."description"`,
	`"model"."model_definition"`,
	`"model"."configuration"`,
	`"model"."visibility"`,
	`"model"."owner"`,
	`"model"."create_time"`,
	`"model"."update_time"`,
}

var UpdateModelSelectedFields = []string{
	`"model"."id"`,
	`"model"."description"`,
	`"model"."update_time"`,
}

func (r *repository) CreateModel(model datamodel.Model) error {
	// We ignore the virtual columns
	if result := r.db.Model(&datamodel.Model{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetModelById(owner string, modelId string) (datamodel.Model, error) {
	var model datamodel.Model
	if result := r.db.Model(&datamodel.Model{}).Select(GetModelSelectedFields).Where(&datamodel.Model{Owner: owner, ID: modelId}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model id %s you specified is not found in namespace %s", modelId, owner)
	}
	return model, nil
}

func (r *repository) ListModel(owner string, view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error) {

	queryBuilder := r.db.Model(&datamodel.Model{}).Where("owner = ?", owner).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		queryBuilder = queryBuilder.Limit(10)
	} else if pageSize > 0 && pageSize <= 100 {
		queryBuilder = queryBuilder.Limit(pageSize)
	} else {
		queryBuilder = queryBuilder.Limit(100)
	}

	if pageToken != "" {
		createTime, uuid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, uuid)
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Model
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createTime = item.CreateTime
		models = append(models, item)
	}

	if len(models) > 0 {
		r.db.Model(&datamodel.Model{}).Where("owner = ?", owner).Count(&totalSize)
		nextPageToken := paginate.EncodeToken(createTime, (models)[len(models)-1].UID.String())
		return models, nextPageToken, totalSize, nil
	}

	return nil, "", 0, nil
}

func (r *repository) UpdateModel(modelUid uuid.UUID, updatedModel datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).Select(UpdateModelSelectedFields).Where("uid", modelUid).Updates(&updatedModel); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) CreateModelInstance(instance datamodel.ModelInstance) error {
	if result := r.db.Model(&datamodel.ModelInstance{}).Create(&instance); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) UpdateModelInstance(modelInstanceUID uuid.UUID, instanceInfo datamodel.ModelInstance) error {
	if result := r.db.Model(&datamodel.ModelInstance{}).Where(map[string]interface{}{"uid": modelInstanceUID}).Updates(&instanceInfo); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetModelInstance(modelUid uuid.UUID, instanceId string) (datamodel.ModelInstance, error) {
	var instanceDB datamodel.ModelInstance
	if result := r.db.Model(&datamodel.ModelInstance{}).Where(map[string]interface{}{"model_uid": modelUid, "id": instanceId}).First(&instanceDB); result.Error != nil {
		return datamodel.ModelInstance{}, status.Errorf(codes.NotFound, "The instance %v for model %v not found", instanceId, modelUid)
	}
	return instanceDB, nil
}

func (r *repository) GetModelInstances(modelUid uuid.UUID) ([]datamodel.ModelInstance, error) {
	var instances []datamodel.ModelInstance
	if result := r.db.Model(&datamodel.ModelInstance{}).Where("model_uid", modelUid).Order("id asc").Find(&instances); result.Error != nil {
		return []datamodel.ModelInstance{}, status.Errorf(codes.NotFound, "The instance for model %v not found", modelUid)
	}
	return instances, nil
}

func (r *repository) ListModelInstance(modelUid uuid.UUID, view modelPB.View, pageSize int, pageToken string) (instances []datamodel.ModelInstance, nextPageToken string, totalSize int64, err error) {

	queryBuilder := r.db.Model(&datamodel.ModelInstance{}).Where("model_uid = ?", modelUid).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		queryBuilder = queryBuilder.Limit(10)
	} else if pageSize > 0 && pageSize <= 100 {
		queryBuilder = queryBuilder.Limit(pageSize)
	} else {
		queryBuilder = queryBuilder.Limit(100)
	}

	if pageToken != "" {
		createTime, uuid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, uuid)
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.ModelInstance
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createTime = item.CreateTime
		instances = append(instances, item)
	}

	if len(instances) > 0 {
		r.db.Model(&datamodel.ModelInstance{}).Where("model_uid = ?", modelUid).Count(&totalSize)
		nextPageToken := paginate.EncodeToken(createTime, (instances)[len(instances)-1].UID.String())
		return instances, nextPageToken, totalSize, nil
	}

	return nil, "", 0, nil
}

func (r *repository) CreateTritonModel(model datamodel.TritonModel) error {
	if result := r.db.Model(&datamodel.TritonModel{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetTritonModels(modelInstanceUID uuid.UUID) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where("model_instance_uid", modelInstanceUID).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model instance id %v not found", modelInstanceUID)
	}
	return tmodels, nil
}

func (r *repository) GetTritonEnsembleModel(modelInstanceUID uuid.UUID) (datamodel.TritonModel, error) {
	var ensembleModel datamodel.TritonModel
	result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_instance_uid": modelInstanceUID, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelInstanceUID)
	}
	return ensembleModel, nil
}

func (r *repository) DeleteModel(modelUid uuid.UUID) error {
	if result := r.db.Select("Instances").Delete(&datamodel.Model{BaseDynamic: datamodel.BaseDynamic{UID: modelUid}}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelUid)
	}
	return nil
}

func (r *repository) GetModelDefinition(id string) (datamodel.ModelDefinition, error) {
	var definitionDB datamodel.ModelDefinition
	if result := r.db.Model(&datamodel.ModelDefinition{}).Where("id", id).First(&definitionDB); result.Error != nil {
		return datamodel.ModelDefinition{}, status.Errorf(codes.NotFound, "The model definition not found")
	}
	return definitionDB, nil
}

func (r *repository) ListModelDefinition(view modelPB.View, pageSize int, pageToken string) (definitions []datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error) {

	queryBuilder := r.db.Model(&datamodel.ModelDefinition{}).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		queryBuilder = queryBuilder.Limit(10)
	} else if pageSize > 0 && pageSize <= 100 {
		queryBuilder = queryBuilder.Limit(pageSize)
	} else {
		queryBuilder = queryBuilder.Limit(100)
	}

	if pageToken != "" {
		createTime, uuid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, uuid)
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.ModelDefinition
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createTime = item.CreateTime
		definitions = append(definitions, item)
	}

	if len(definitions) > 0 {
		r.db.Model(&datamodel.ModelDefinition{}).Count(&totalSize)
		nextPageToken := paginate.EncodeToken(createTime, (definitions)[len(definitions)-1].UID.String())
		return definitions, nextPageToken, totalSize, nil
	}

	return nil, "", 0, nil
}
