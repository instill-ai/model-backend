package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/x/errmsg"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

const VisibilityPublic = datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PUBLIC)

type Repository interface {
	ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error)
	GetModelByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, includeAvatar bool) (*datamodel.Model, error)

	CreateNamespaceModel(ctx context.Context, ownerPermalink string, model *datamodel.Model) error
	ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error)
	GetNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, includeAvatar bool) (*datamodel.Model, error)

	UpdateNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, model *datamodel.Model) error
	UpdateNamespaceModelIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error
	DeleteNamespaceModelByID(ctx context.Context, ownerPermalink string, id string) error

	GetModelDefinition(id string) (*datamodel.ModelDefinition, error)
	GetModelDefinitionByUID(uid uuid.UUID) (*datamodel.ModelDefinition, error)
	ListModelDefinitions(view modelpb.View, pageSize int64, pageToken string) (definitions []*datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error)

	GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool, includeAvatar bool) (*datamodel.Model, error)
	ListModelsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Model, int64, string, error)

	CreateModelVersion(ctx context.Context, ownerPermalink string, version *datamodel.ModelVersion) error
	UpdateModelVersionDigestByID(ctx context.Context, modelUID uuid.UUID, versionID string, digest string) error
	GetModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) (version *datamodel.ModelVersion, err error)
	DeleteModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) error
	DeleteModelVersionByDigest(ctx context.Context, modelUID uuid.UUID, digest string) error
	GetLatestModelVersionByModelUID(ctx context.Context, modelUID uuid.UUID) (version *datamodel.ModelVersion, err error)
	ListModelVersions(ctx context.Context, modelUID uuid.UUID) (versions []*datamodel.ModelVersion, err error)
	ListModelVersionsByDigest(ctx context.Context, modelUID uuid.UUID, digest string) (versions []*datamodel.ModelVersion, err error)

	CreateModelTag(ctx context.Context, modelUID uuid.UUID, tagName string) error
	DeleteModelTag(ctx context.Context, modelUID uuid.UUID, tagName string) error
	ListModelTags(ctx context.Context, modelUID uuid.UUID) ([]*datamodel.ModelTag, error)
}

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

type repository struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// NewRepository initiates a repository instance
func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	return &repository{
		db:          db,
		redisClient: redisClient,
	}
}
func (r *repository) checkPinnedUser(ctx context.Context, db *gorm.DB, table string) *gorm.DB {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// If the user is pinned, we will use the primary database for querying.
	if !errors.Is(r.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table)).Err(), redis.Nil) {
		db = db.Clauses(dbresolver.Write)
	}
	return db
}

func (r *repository) pinUser(ctx context.Context, table string) {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// To solve the read-after-write inconsistency problem,
	// we will direct the user to read from the primary database for a certain time frame
	// to ensure that the data is synchronized from the primary DB to the replica DB.
	_ = r.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table), time.Now(), time.Duration(config.Config.Database.Replica.ReplicationTimeFrame)*time.Second)
}

func (r *repository) listModels(ctx context.Context, where string, whereArgs []any, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", err
	}
	if expr != nil {
		if len(whereArgs) == 0 {
			where = "(?)"
			whereArgs = append(whereArgs, expr)
		} else {
			where = fmt.Sprintf("((%s) AND ?)", where)
			whereArgs = append(whereArgs, expr)
		}
	}

	joinStr := "left join model_tag on model_tag.model_uid = model.uid"

	countBuilder := db.Model(&datamodel.Model{}).Where(where, whereArgs...).Joins(joinStr)
	if uidAllowList != nil {
		countBuilder = countBuilder.Where("uid in ?", uidAllowList).Count(&totalSize)
	}

	countBuilder.Count(&totalSize)

	queryBuilder := db.Model(&datamodel.Model{}).Joins(joinStr).Where(where, whereArgs...)
	if order.Fields == nil || len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "create_time",
			Desc: true,
		})
	}
	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}
	queryBuilder.Order("uid DESC")

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
		tokens, err := DecodeToken(pageToken)
		if err != nil {
			logger.Error(err.Error())
			return nil, 0, "", ErrPageTokenDecode
		}

		for _, o := range order.Fields {

			p := strcase.ToSnake(o.Path)
			if v, ok := tokens[p]; ok {
				switch p {
				case "create_time", "update_time":
					// Add "model." prefix to prevent ambiguous since tag table also has the two columns.
					if o.Desc {
						queryBuilder = queryBuilder.Where("model."+p+" < ?::timestamp", v)
					} else {
						queryBuilder = queryBuilder.Where("model."+p+" > ?::timestamp", v)
					}
				default:
					if o.Desc {
						queryBuilder = queryBuilder.Where(p+" < ?", v)
					} else {
						queryBuilder = queryBuilder.Where(p+" > ?", v)
					}
				}
			}
		}
	}

	if isBasicView {
		queryBuilder.Omit("configuration")
	}
	queryBuilder.Omit("profile_image")

	result := queryBuilder.Preload("Tags").Find(&models)
	if result.Error != nil {
		logger.Error(result.Error.Error())
		return nil, 0, "", result.Error
	}

	if len(models) > 0 {
		lastID := (models)[len(models)-1].ID
		lastUID := (models)[len(models)-1].UID
		lastCreateTime := (models)[len(models)-1].CreateTime
		lastUpdateTime := (models)[len(models)-1].UpdateTime
		lastItem := &datamodel.Model{}

		tokens := map[string]string{}

		lastItemQueryBuilder := db.Model(&datamodel.Model{}).Joins(joinStr).Omit("profile_image").Where(where, whereArgs...)
		if uidAllowList != nil {
			lastItemQueryBuilder = lastItemQueryBuilder.Where("uid in ?", uidAllowList)

		}

		for _, field := range order.Fields {
			orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(!field.Desc)
			lastItemQueryBuilder.Order(orderString)
			switch p := strcase.ToSnake(field.Path); p {
			case "id":
				tokens[p] = lastID
			case "create_time":
				tokens[p] = lastCreateTime.Format(time.RFC3339Nano)
			case "update_time":
				tokens[p] = lastUpdateTime.Format(time.RFC3339Nano)
			}

		}
		lastItemQueryBuilder.Order("uid ASC")
		tokens["uid"] = lastUID.String()

		if result := lastItemQueryBuilder.Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", err
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken, err = EncodeToken(tokens)
			if err != nil {
				return nil, 0, "", err
			}
		}
	}

	return models, totalSize, nextPageToken, nil
}

func (r *repository) ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {
	models, totalSize, nextPageToken, err = r.listModels(ctx,
		"",
		[]any{},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, order)

	return models, totalSize, nextPageToken, err
}

func (r *repository) ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {
	models, totalSize, nextPageToken, err = r.listModels(ctx,
		"(owner = ?)",
		[]any{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, order)

	return models, totalSize, nextPageToken, err
}

func (r *repository) ListModelsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Model, int64, string, error) {
	return r.listModels(ctx, "", []any{}, pageSize, pageToken, isBasicView, filter, nil, showDeleted, ordering.OrderBy{})
}

func (r *repository) getNamespaceModel(ctx context.Context, where string, whereArgs []any, isBasicView bool, includeAvatar bool) (*datamodel.Model, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	db := r.checkPinnedUser(ctx, r.db, "model")

	var model datamodel.Model

	queryBuilder := db.Model(&datamodel.Model{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}
	if !includeAvatar {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&model); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getNamespaceModel error: %s", result.Error.Error()),
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

func (r *repository) GetModelByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, includeAvatar bool) (*datamodel.Model, error) {
	// TODO: ACL
	return r.getNamespaceModel(ctx,
		"(uid = ?)",
		[]any{uid},
		isBasicView,
		includeAvatar)
}

func (r *repository) GetNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, includeAvatar bool) (*datamodel.Model, error) {
	return r.getNamespaceModel(ctx,
		"(id = ? AND owner = ? )",
		[]any{id, ownerPermalink},
		isBasicView,
		includeAvatar)
}

func (r *repository) GetModelByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool, includeAvatar bool) (*datamodel.Model, error) {
	return r.getNamespaceModel(ctx,
		"(uid = ?)",
		[]any{uid},
		isBasicView,
		includeAvatar)
}

func (r *repository) CreateNamespaceModel(ctx context.Context, ownerPermalink string, model *datamodel.Model) error {

	r.pinUser(ctx, "model")
	db := r.checkPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).Create(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *repository) UpdateNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, model *datamodel.Model) error {

	r.pinUser(ctx, "model")
	db := r.checkPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Updates(model); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) UpdateNamespaceModelIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {

	r.pinUser(ctx, "model")
	db := r.checkPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespaceModelByID(ctx context.Context, ownerPermalink string, id string) error {

	r.pinUser(ctx, "model")
	db := r.checkPinnedUser(ctx, r.db, "model")

	result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Delete(&datamodel.Model{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) CreateModelVersion(ctx context.Context, ownerPermalink string, version *datamodel.ModelVersion) error {

	r.pinUser(ctx, "model_version")
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).Create(&version); result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) UpdateModelVersionDigestByID(ctx context.Context, modelUID uuid.UUID, versionID string, digest string) error {

	r.pinUser(ctx, "model_version")
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).
		Where("(version = ? AND model_uid = ?)", versionID, modelUID).
		Update("digest", digest); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}

	return nil
}

func (r *repository) DeleteModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) error {

	r.pinUser(ctx, "model_version")
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	result := db.Model(&datamodel.ModelVersion{}).
		Where("(version = ? AND model_uid = ?)", versionID, modelUID).
		Delete(&datamodel.ModelVersion{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) DeleteModelVersionByDigest(ctx context.Context, modelUID uuid.UUID, digest string) error {

	r.pinUser(ctx, "model_version")
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	result := db.Model(&datamodel.ModelVersion{}).
		Where("(digest = ? AND model_uid = ?)", digest, modelUID).
		Delete(&datamodel.ModelVersion{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) GetLatestModelVersionByModelUID(ctx context.Context, modelUID uuid.UUID) (version *datamodel.ModelVersion, err error) {
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	queryBuilder := db.Model(&datamodel.ModelVersion{}).Where("(model_uid = ?)", modelUID)

	if result := queryBuilder.Order("update_time DESC").First(&version); result.Error != nil {
		st, _ := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] GetModelVersionByName error: %s", result.Error.Error()),
			"model",
			"",
			"",
			result.Error.Error(),
		)
		return nil, st.Err()
	}
	return version, nil
}

func (r *repository) GetModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) (version *datamodel.ModelVersion, err error) {
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	queryBuilder := db.Model(&datamodel.ModelVersion{}).Where("(version = ? AND model_uid = ?)", versionID, modelUID)

	if result := queryBuilder.First(&version); result.Error != nil {
		st, _ := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] GetModelVersionByName error: %s", result.Error.Error()),
			"model",
			"",
			"",
			result.Error.Error(),
		)
		return nil, st.Err()
	}
	return version, nil
}

func (r *repository) ListModelVersionsByDigest(ctx context.Context, modelUID uuid.UUID, digest string) (versions []*datamodel.ModelVersion, err error) {
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).Where("(digest = ? AND model_uid = ?)", digest, modelUID).Find(&versions); result.Error != nil {
		return []*datamodel.ModelVersion{}, status.Errorf(codes.NotFound, "The model versions belongs to model uid %v with digest %s not found", modelUID, digest)
	}
	return versions, nil
}

func (r *repository) ListModelVersions(ctx context.Context, modelUID uuid.UUID) (versions []*datamodel.ModelVersion, err error) {
	db := r.checkPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).Where("model_uid", modelUID).Find(&versions); result.Error != nil {
		return []*datamodel.ModelVersion{}, status.Errorf(codes.NotFound, "The model versions belongs to model uid %v not found", modelUID)
	}
	return versions, nil
}

func (r *repository) CreateModelTag(ctx context.Context, modelUID uuid.UUID, tagName string) error {

	r.pinUser(ctx, "model_tag")

	db := r.checkPinnedUser(ctx, r.db, "model_tag")

	tag := datamodel.ModelTag{
		ModelUID: modelUID.String(),
		TagName:  tagName,
	}

	if result := db.Model(&datamodel.ModelTag{}).Create(&tag); result.Error != nil {

		var pgErr *pgconn.PgError

		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {

			return errmsg.AddMessage(ErrNameExists, "Tag already exists")

		}

		return result.Error

	}

	return nil

}

func (r *repository) DeleteModelTag(ctx context.Context, modelUID uuid.UUID, tagName string) error {

	r.pinUser(ctx, "model_tag")

	db := r.checkPinnedUser(ctx, r.db, "model_tag")

	result := db.Model(&datamodel.ModelTag{}).Where("model_uid = ? and tag_name = ?", modelUID, tagName).Delete(&datamodel.ModelTag{})

	if result.Error != nil {

		return result.Error

	}

	if result.RowsAffected == 0 {

		return ErrNoDataDeleted

	}

	return nil

}

func (r *repository) ListModelTags(ctx context.Context, modelUID uuid.UUID) ([]*datamodel.ModelTag, error) {

	db := r.checkPinnedUser(ctx, r.db, "model_tag")

	var tags []*datamodel.ModelTag

	result := db.Model(&datamodel.ModelTag{}).Where("model_uid = ?", modelUID).Find(tags)

	if result.Error != nil {

		return nil, result.Error

	}

	return tags, nil

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

func (r *repository) ListModelDefinitions(view modelpb.View, pageSize int64, pageToken string) (definitions []*datamodel.ModelDefinition, nextPageToken string, totalSize int64, err error) {
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

	if view != modelpb.View_VIEW_FULL {
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

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses
func (r *repository) transpileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&Transpiler{
		filter: filter,
	}).Transpile()
}
