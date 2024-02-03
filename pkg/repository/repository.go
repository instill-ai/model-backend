package repository

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/x/paginate"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

const VisibilityPublic = datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC)

type Repository interface {
	ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Model, int64, string, error)
	GetModelByUID(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Model, error)

	CreateNamespaceModel(ctx context.Context, ownerPermalink string, model *datamodel.Model) error
	ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Model, int64, string, error)
	GetNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Model, error)

	UpdateNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, model *datamodel.Model) error
	UpdateNamespaceModelIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error
	UpdateNamespaceModelStateByID(ctx context.Context, ownerPermalink string, id string, state *datamodel.ModelState) error
	DeleteNamespaceModelByID(ctx context.Context, ownerPermalink string, id string) error

	CreateInferenceModel(ctx context.Context, ownerPermalink string, model *datamodel.InferenceModel) error
	GetInferenceModels(ctx context.Context, modelUID uuid.UUID) ([]*datamodel.InferenceModel, error)
	GetInferenceEnsembleModel(ctx context.Context, modelUID uuid.UUID) (*datamodel.InferenceModel, error)

	GetModelDefinition(id string) (*datamodel.ModelDefinition, error)
	GetModelDefinitionByUID(uid uuid.UUID) (*datamodel.ModelDefinition, error)
	ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) (definitions []*datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error)

	GetModelByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Model, error)
	GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Model, error)
	ListModelsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, showDeleted bool) ([]*datamodel.Model, int64, string, error)
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

func (r *repository) listModels(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, uidAllowList []uuid.UUID, showDeleted bool) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	if uidAllowList != nil {
		db.Model(&datamodel.Model{}).Where(where, whereArgs...).Where("uid in ?", uidAllowList).Count(&totalSize)
	} else {
		db.Model(&datamodel.Model{}).Where(where, whereArgs...).Count(&totalSize)
	}

	queryBuilder := db.Model(&datamodel.Model{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

	if uidAllowList != nil {
		queryBuilder = queryBuilder.Where("uid in ?", uidAllowList)
	}

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			logger.Error(err.Error())
			return nil, 0, "", ErrPageTokenDecode
		}

		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		logger.Error(err.Error())
		return nil, 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Model
		if err = db.ScanRows(rows, &item); err != nil {
			logger.Error(err.Error())
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		models = append(models, &item)
	}

	if len(models) > 0 {
		lastUID := (models)[len(models)-1].UID
		lastItem := &datamodel.Model{}

		if uidAllowList != nil {
			if result := db.Model(&datamodel.Model{}).
				Where(where, whereArgs...).
				Where("uid in ?", uidAllowList).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				logger.Error(err.Error())
				return nil, 0, "", err
			}
		} else {
			if result := db.Model(&datamodel.Model{}).
				Where(where, whereArgs...).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				logger.Error(err.Error())
				return nil, 0, "", err
			}
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return models, totalSize, nextPageToken, nil
}

func (r *repository) ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Model, int64, string, error) {
	return r.listModels(ctx,
		"",
		[]interface{}{},
		pageSize, pageToken, isBasicView, uidAllowList, showDeleted)
}

func (r *repository) ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Model, int64, string, error) {
	return r.listModels(ctx,
		"(owner = ?)",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, uidAllowList, showDeleted)
}

func (r *repository) ListModelsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, showDeleted bool) ([]*datamodel.Model, int64, string, error) {
	return r.listModels(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, nil, showDeleted)
}

func (r *repository) getNamespaceModel(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.Model, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var model datamodel.Model

	queryBuilder := r.db.Model(&datamodel.Model{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&model); result.Error != nil {
		logger.Error(result.Error.Error())
		return nil, result.Error
	}
	return &model, nil
}

func (r *repository) GetModelByUID(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Model, error) {
	// TODO: ACL
	return r.getNamespaceModel(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView)
}

func (r *repository) GetNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Model, error) {
	return r.getNamespaceModel(ctx,
		"(id = ? AND owner = ? )",
		[]interface{}{id, ownerPermalink},
		isBasicView)
}

func (r *repository) GetModelByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Model, error) {
	return r.getNamespaceModel(ctx,
		"(id = ?)",
		[]interface{}{id},
		isBasicView)
}

func (r *repository) GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Model, error) {
	return r.getNamespaceModel(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView,
	)
}

func (r *repository) CreateNamespaceModel(ctx context.Context, ownerPermalink string, model *datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).Create(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *repository) UpdateNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, model *datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Updates(model); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) UpdateNamespaceModelIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {
	if result := r.db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

// TODO: gorm do not update the zero value with struct, so we need to update the state manually.
func (r *repository) UpdateNamespaceModelStateByID(ctx context.Context, ownerPermalink string, id string, state *datamodel.ModelState) error {
	if result := r.db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Updates(map[string]interface{}{"state": state}); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}

	return nil
}

func (r *repository) DeleteNamespaceModelByID(ctx context.Context, ownerPermalink string, id string) error {
	if result := r.db.Select("InferenceModels").Delete(&datamodel.Model{ID: id, Owner: ownerPermalink}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %s", id)
	}
	return nil
}

func (r *repository) CreateInferenceModel(ctx context.Context, ownerPermalink string, model *datamodel.InferenceModel) error {
	if result := r.db.Model(&datamodel.InferenceModel{}).Create(&model); result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) GetInferenceModels(ctx context.Context, modelUID uuid.UUID) ([]*datamodel.InferenceModel, error) {
	var models []*datamodel.InferenceModel
	if result := r.db.Model(&datamodel.InferenceModel{}).Where("model_uid", modelUID).Find(&models); result.Error != nil {
		return []*datamodel.InferenceModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelUID)
	}
	return models, nil
}

func (r *repository) GetInferenceEnsembleModel(ctx context.Context, modelUID uuid.UUID) (*datamodel.InferenceModel, error) {
	var ensembleModel *datamodel.InferenceModel
	result := r.db.Model(&datamodel.InferenceModel{}).
		Where("(model_uid = ? AND (platform = ? OR platform = ?))", modelUID, "ensemble", "ray").
		First(&ensembleModel)
	if result.Error != nil {
		return &datamodel.InferenceModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelUID)
	}
	return ensembleModel, nil
}

func (r *repository) GetModelDefinition(id string) (*datamodel.ModelDefinition, error) {
	var definitionDB *datamodel.ModelDefinition
	if result := r.db.Model(&datamodel.ModelDefinition{}).Where("id", id).First(&definitionDB); result.Error != nil {
		return &datamodel.ModelDefinition{}, status.Errorf(codes.NotFound, "The model definition not found")
	}
	return definitionDB, nil
}

func (r *repository) GetModelDefinitionByUID(uid uuid.UUID) (*datamodel.ModelDefinition, error) {
	var definitionDB *datamodel.ModelDefinition
	if result := r.db.Model(&datamodel.ModelDefinition{}).Where("uid", uid).First(&definitionDB); result.Error != nil {
		return &datamodel.ModelDefinition{}, status.Errorf(codes.NotFound, "The model definition not found")
	}
	return definitionDB, nil
}

func (r *repository) ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) (definitions []*datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error) {
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
		var item *datamodel.ModelDefinition
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
