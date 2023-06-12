package repository

import (
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	"github.com/instill-ai/x/paginate"
)

type Repository interface {
	CreateModel(model datamodel.Model) error
	CreatePreDeployModel(model datamodel.PreDeployModel) error
	GetModelByID(owner string, modelID string, view modelPB.View) (datamodel.Model, error)
	GetModelByUID(owner string, modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error)
	DeleteModel(modelUID uuid.UUID) error
	UpdateModel(modelUID uuid.UUID, updatedModel datamodel.Model) error
	UpdateModelState(modelUID uuid.UUID, state datamodel.ModelState) error
	ListModels(owner string, view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error)

	CreateTritonModel(model datamodel.TritonModel) error
	GetTritonModels(modelUID uuid.UUID) ([]datamodel.TritonModel, error)
	GetTritonEnsembleModel(modelUID uuid.UUID) (datamodel.TritonModel, error)
	GetModelDefinition(id string) (datamodel.ModelDefinition, error)
	GetModelDefinitionByUID(uid uuid.UUID) (datamodel.ModelDefinition, error)
	ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) (definitions []datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error)

	GetModelByIDAdmin(modelID string, view modelPB.View) (datamodel.Model, error)
	GetModelByUIDAdmin(modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error)
	ListModelsAdmin(view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error)
}

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

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
	`"model"."model_definition_uid"`,
	`"model"."configuration"`,
	`"model"."visibility"`,
	`"model"."owner"`,
	`"model"."state"`,
	`"model"."task"`,
	`"model"."create_time"`,
	`"model"."update_time"`,
}

var GetModelSelectedFieldsWOConfiguration = []string{
	`CONCAT('models/', id) as name`,
	`"model"."uid"`,
	`"model"."id"`,
	`"model"."description"`,
	`"model"."model_definition_uid"`,
	`"model"."visibility"`,
	`"model"."owner"`,
	`"model"."state"`,
	`"model"."task"`,
	`"model"."create_time"`,
	`"model"."update_time"`,
}

func (r *repository) CreateModel(model datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) CreatePreDeployModel(model datamodel.PreDeployModel) error {
	if result := r.db.Model(&datamodel.Model{}).Create(&model); result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) GetModelByID(owner string, modelID string, view modelPB.View) (datamodel.Model, error) {
	var model datamodel.Model
	selectedFields := GetModelSelectedFields
	if view != modelPB.View_VIEW_FULL {
		selectedFields = GetModelSelectedFieldsWOConfiguration
	}
	if result := r.db.Model(&datamodel.Model{}).Select(selectedFields).Where(&datamodel.Model{Owner: owner, ID: modelID}).
		Or(&datamodel.Model{Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC), ID: modelID}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model id %s you specified is not found in namespace %s", modelID, owner)
	}
	return model, nil
}

func (r *repository) GetModelByIDAdmin(modelID string, view modelPB.View) (datamodel.Model, error) {
	var model datamodel.Model
	selectedFields := GetModelSelectedFields
	if view != modelPB.View_VIEW_FULL {
		selectedFields = GetModelSelectedFieldsWOConfiguration
	}
	if result := r.db.Model(&datamodel.Model{}).Select(selectedFields).Where(&datamodel.Model{ID: modelID}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model id %s you specified is not found", modelID)
	}
	return model, nil
}

func (r *repository) GetModelByUID(owner string, modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error) {
	var model datamodel.Model
	selectedFields := GetModelSelectedFields
	if view != modelPB.View_VIEW_FULL {
		selectedFields = GetModelSelectedFieldsWOConfiguration
	}
	if result := r.db.Model(&datamodel.Model{}).Select(selectedFields).Where(&datamodel.Model{Owner: owner, BaseDynamic: datamodel.BaseDynamic{UID: modelUID}}).
		Or(&datamodel.Model{Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC), BaseDynamic: datamodel.BaseDynamic{UID: modelUID}}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model uid %s you specified is not found in namespace %s", modelUID, owner)
	}
	return model, nil
}

func (r *repository) GetModelByUIDAdmin(modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error) {
	var model datamodel.Model
	selectedFields := GetModelSelectedFields
	if view != modelPB.View_VIEW_FULL {
		selectedFields = GetModelSelectedFieldsWOConfiguration
	}
	if result := r.db.Model(&datamodel.Model{}).Select(selectedFields).Where(&datamodel.Model{BaseDynamic: datamodel.BaseDynamic{UID: modelUID}}).First(&model); result.Error != nil {
		return datamodel.Model{}, status.Errorf(codes.NotFound, "The model uid %s you specified is not found", modelUID)
	}
	return model, nil
}

func (r *repository) ListModels(owner string, view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error) {
	if result := r.db.Model(&datamodel.Model{}).Where("owner = ? or visibility = ?", owner, datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC)).Count(&totalSize); result.Error != nil {
		return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.Model{}).Order("create_time DESC, id DESC").Where("owner = ? or visibility = ?", owner, datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC))

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, id, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, id)
	}

	if view != modelPB.View_VIEW_FULL {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Model
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		models = append(models, item)
	}

	if len(models) > 0 {
		lastID := (models)[len(models)-1].ID
		lastItem := &datamodel.Model{}
		if result := r.db.Model(&datamodel.Model{}).
			Where("owner = ? or visibility = ?", owner, datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC)).
			Order("create_time ASC, id ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.ID == lastID {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastID)
		}
	}

	return models, nextPageToken, totalSize, nil
}

func (r *repository) ListModelsAdmin(view modelPB.View, pageSize int, pageToken string) (models []datamodel.Model, nextPageToken string, totalSize int64, err error) {
	if result := r.db.Model(&datamodel.Model{}).Count(&totalSize); result.Error != nil {
		return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.Model{}).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, id, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, id)
	}

	if view != modelPB.View_VIEW_FULL {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Model
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		models = append(models, item)
	}

	if len(models) > 0 {
		lastID := (models)[len(models)-1].ID
		lastItem := &datamodel.Model{}
		if result := r.db.Model(&datamodel.Model{}).
			Order("create_time ASC, id ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.ID == lastID {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastID)
		}
	}

	return models, nextPageToken, totalSize, nil
}

func (r *repository) UpdateModel(modelUID uuid.UUID, updatedModel datamodel.Model) error {
	result := r.db.Model(&datamodel.Model{}).Where("uid", modelUID).Updates(&updatedModel)
	if result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

// TODO: gorm do not update the zero value with struct, so we need to update the state manually.
func (r *repository) UpdateModelState(modelUID uuid.UUID, state datamodel.ModelState) error {
	if result := r.db.Model(&datamodel.Model{}).Where(map[string]interface{}{"uid": modelUID}).Updates(map[string]interface{}{"state": state}); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) CreateTritonModel(model datamodel.TritonModel) error {
	if result := r.db.Model(&datamodel.TritonModel{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetTritonModels(modelUID uuid.UUID) ([]datamodel.TritonModel, error) {
	var tmodels []datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where("model_uid", modelUID).Find(&tmodels); result.Error != nil {
		return []datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelUID)
	}
	return tmodels, nil
}

func (r *repository) GetTritonEnsembleModel(modelUID uuid.UUID) (datamodel.TritonModel, error) {
	var ensembleModel datamodel.TritonModel
	result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_uid": modelUID, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelUID)
	}
	return ensembleModel, nil
}

func (r *repository) DeleteModel(modelUID uuid.UUID) error {
	if result := r.db.Select("TritonModels").Delete(&datamodel.Model{BaseDynamic: datamodel.BaseDynamic{UID: modelUID}}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelUID)
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

func (r *repository) GetModelDefinitionByUID(uid uuid.UUID) (datamodel.ModelDefinition, error) {
	var definitionDB datamodel.ModelDefinition
	if result := r.db.Model(&datamodel.ModelDefinition{}).Where("uid", uid).First(&definitionDB); result.Error != nil {
		return datamodel.ModelDefinition{}, status.Errorf(codes.NotFound, "The model definition not found")
	}
	return definitionDB, nil
}

func (r *repository) ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) (definitions []datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error) {
	if result := r.db.Model(&datamodel.ModelDefinition{}).Count(&totalSize); result.Error != nil {
		return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.ModelDefinition{}).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, id, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", 0, status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, id)
	}

	if view != modelPB.View_VIEW_FULL {
		queryBuilder.Omit("model_spec")
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.ModelDefinition
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", 0, status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		definitions = append(definitions, item)
	}

	if len(definitions) > 0 {
		lastID := (definitions)[len(definitions)-1].ID
		lastItem := &datamodel.ModelDefinition{}
		if result := r.db.Model(&datamodel.ModelDefinition{}).
			Order("create_time ASC, id ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.ID == lastID {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastID)
		}
	}

	return definitions, nextPageToken, totalSize, nil
}
