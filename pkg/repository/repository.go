package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

const VisibilityPublic = datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC)

type Repository interface {
	ListModels(ctx context.Context, userPermalink string, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error)
	CreatePreDeployModel(model *datamodel.PreDeployModel) error
	GetModelByUID(ctx context.Context, userPermalink string, view modelPB.View, uid uuid.UUID) (*datamodel.Model, error)

	CreateUserModel(model *datamodel.Model) error
	ListUserModels(ctx context.Context, ownerPermalink string, userPermalink string, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error)
	GetUserModelByID(ctx context.Context, ownerPermalink string, userPermalink string, modelID string, view modelPB.View) (*datamodel.Model, error)
	UpdateUserModel(ownerPermalink string, userPermalink string, modelUID uuid.UUID, updatedModel *datamodel.Model) error
	UpdateUserModelState(ownerPermalink string, userPermalink string, modelUID uuid.UUID, state *datamodel.ModelState) error

	CreateTritonModel(model *datamodel.TritonModel) error
	GetTritonModels(modelUID uuid.UUID) ([]*datamodel.TritonModel, error)
	GetTritonEnsembleModel(modelUID uuid.UUID) (*datamodel.TritonModel, error)

	GetModelDefinition(id string) (*datamodel.ModelDefinition, error)
	GetModelDefinitionByUID(uid uuid.UUID) (*datamodel.ModelDefinition, error)
	ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) (definitions []*datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error)
	DeleteModel(modelUID uuid.UUID) error

	GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*datamodel.Model, error)
	GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, view modelPB.View) (*datamodel.Model, error)
	ListModelsAdmin(ctx context.Context, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error)
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

func (r *repository) listModels(ctx context.Context, where string, whereArgs []interface{}, view modelPB.View, pageSize int, pageToken string, showDeleted bool) (models []*datamodel.Model, nextPageToken string, totalSize int64, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	if result := db.Model(&datamodel.Model{}).Where(where, whereArgs...).Count(&totalSize); result.Error != nil {
		logger.Error(result.Error.Error())
		return nil, "", 0, status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := db.Model(&datamodel.Model{}).Order("create_time DESC, id DESC").Where(where, whereArgs...)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list Model error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, "", 0, st.Err()
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createdAt, uid)
	}

	if view != modelPB.View_VIEW_FULL {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list Model error: %s", err.Error()),
			"model",
			"",
			"",
			err.Error(),
		)

		if err != nil {
			logger.Error(err.Error())
		}
		return nil, "", 0, st.Err()
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Model
		if err = db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list Model error: %s", err.Error()),
				"model",
				"",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, "", 0, st.Err()
		}
		createTime = item.CreateTime
		models = append(models, &item)
	}

	if len(models) > 0 {
		lastID := (models)[len(models)-1].ID
		lastItem := &datamodel.Model{}
		if result := db.Model(&datamodel.Model{}).
			Where(where, whereArgs...).
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

func (r *repository) ListModels(ctx context.Context, userPermalink string, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error) {
	return r.listModels(ctx,
		"(owner = ? OR visibility = ?)",
		[]interface{}{userPermalink, VisibilityPublic},
		view, pageSize, pageToken, showDeleted,
	)
}

func (r *repository) ListUserModels(ctx context.Context, ownerPermalink string, userPermalink string, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error) {
	return r.listModels(ctx,
		"(owner = ? AND (visibility = ? OR ? = ?))",
		[]interface{}{ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		view, pageSize, pageToken, showDeleted,
	)
}

func (r *repository) ListModelsAdmin(ctx context.Context, view modelPB.View, pageSize int, pageToken string, showDeleted bool) ([]*datamodel.Model, string, int64, error) {
	return r.listModels(ctx, "", []interface{}{}, view, pageSize, pageToken, showDeleted)
}

func (r *repository) getUserModel(ctx context.Context, where string, whereArgs []interface{}, view modelPB.View) (*datamodel.Model, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var model datamodel.Model

	queryBuilder := r.db.Model(&datamodel.Model{}).Where(where, whereArgs...)

	if view != modelPB.View_VIEW_FULL {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&model); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getUserModel error: %s", result.Error.Error()),
			"model",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &model, nil
}

func (r *repository) GetModelByUID(ctx context.Context, userPermalink string, view modelPB.View, uid uuid.UUID) (*datamodel.Model, error) {
	return r.getUserModel(ctx,
		"(uid = ? AND (visibility = ? OR owner = ?))",
		[]interface{}{uid, VisibilityPublic, userPermalink},
		view,
	)
}

func (r *repository) GetUserModelByID(ctx context.Context, ownerPermalink string, userPermalink string, modelID string, view modelPB.View) (*datamodel.Model, error) {
	return r.getUserModel(ctx,
		"(id = ? AND (owner = ? AND (visibility = ? OR ? = ?)))",
		[]interface{}{modelID, ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		view,
	)
}

func (r *repository) GetModelByIDAdmin(ctx context.Context, modelID string, view modelPB.View) (*datamodel.Model, error) {
	return r.getUserModel(ctx,
		"(id = ?)",
		[]interface{}{modelID},
		view,
	)
}

func (r *repository) GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, view modelPB.View) (*datamodel.Model, error) {
	return r.getUserModel(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		view,
	)
}

func (r *repository) CreateUserModel(model *datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).Create(model); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				return status.Errorf(codes.AlreadyExists, pgErr.Message)
			}
		}
	}

	return nil
}

func (r *repository) CreatePreDeployModel(model *datamodel.PreDeployModel) error {
	if result := r.db.Model(&datamodel.Model{}).Create(&model); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				return status.Errorf(codes.AlreadyExists, pgErr.Message)
			}
		}
	}

	return nil
}

func (r *repository) UpdateUserModel(ownerPermalink string, userPermalink string, modelUID uuid.UUID, updatedModel *datamodel.Model) error {
	if result := r.db.Model(&datamodel.Model{}).
		Where("(uid = ? AND owner = ? AND ? = ? )", modelUID, ownerPermalink, ownerPermalink, userPermalink).
		Updates(updatedModel); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdateModel] The model uid %s you specified is not found", modelUID)
	}
	return nil
}

// TODO: gorm do not update the zero value with struct, so we need to update the state manually.
func (r *repository) UpdateUserModelState(ownerPermalink string, userPermalink string, modelUID uuid.UUID, state *datamodel.ModelState) error {
	if result := r.db.Model(&datamodel.Model{}).
		Where("(uid = ? AND owner = ? AND ? = ? )", modelUID, ownerPermalink, ownerPermalink, userPermalink).
		Updates(map[string]interface{}{"state": state}); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) CreateTritonModel(model *datamodel.TritonModel) error {
	if result := r.db.Model(&datamodel.TritonModel{}).Create(&model); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *repository) GetTritonModels(modelUID uuid.UUID) ([]*datamodel.TritonModel, error) {
	var tmodels []*datamodel.TritonModel
	if result := r.db.Model(&datamodel.TritonModel{}).Where("model_uid", modelUID).Find(&tmodels); result.Error != nil {
		return []*datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton model belongs to model id %v not found", modelUID)
	}
	return tmodels, nil
}

func (r *repository) GetTritonEnsembleModel(modelUID uuid.UUID) (*datamodel.TritonModel, error) {
	var ensembleModel *datamodel.TritonModel
	result := r.db.Model(&datamodel.TritonModel{}).Where(map[string]interface{}{"model_uid": modelUID, "platform": "ensemble"}).First(&ensembleModel)
	if result.Error != nil {
		return &datamodel.TritonModel{}, status.Errorf(codes.NotFound, "The Triton ensemble model belongs to model id %v not found", modelUID)
	}
	return ensembleModel, nil
}

func (r *repository) DeleteModel(modelUID uuid.UUID) error {
	if result := r.db.Select("TritonModels").Delete(&datamodel.Model{BaseDynamic: datamodel.BaseDynamic{UID: modelUID}}); result.Error != nil {
		return status.Errorf(codes.NotFound, "Could not delete model with id %v", modelUID)
	}
	return nil
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
