package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/paginate"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	errorsx "github.com/instill-ai/x/errors"
	logx "github.com/instill-ai/x/log"
	resourcex "github.com/instill-ai/x/resource"
)

type Repository interface {
	PinUser(ctx context.Context, table string)
	CheckPinnedUser(ctx context.Context, db *gorm.DB, table string) *gorm.DB

	ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy, visibility *modelpb.Model_Visibility) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error)
	GetModelByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, includeAvatar bool) (*datamodel.Model, error)

	CreateNamespaceModel(ctx context.Context, ownerPermalink string, model *datamodel.Model) error
	ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy, visibility *modelpb.Model_Visibility) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error)
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
	ListModelVersions(ctx context.Context, modelUID uuid.UUID, groupDigest bool) (versions []*datamodel.ModelVersion, err error)
	ListModelVersionsByDigest(ctx context.Context, modelUID uuid.UUID, digest string) (versions []*datamodel.ModelVersion, err error)

	CreateModelTags(ctx context.Context, modelUID uuid.UUID, tagNames []string) error
	DeleteModelTags(ctx context.Context, modelUID uuid.UUID, tagNames []string) error
	ListModelTags(ctx context.Context, modelUID uuid.UUID) ([]datamodel.ModelTag, error)

	GetModelRunByUID(ctx context.Context, triggerUID string) (modelRun *datamodel.ModelRun, err error)
	GetLatestModelRunByModelUID(ctx context.Context, userUID string, modelUID string) (modelRun *datamodel.ModelRun, err error)
	GetLatestModelVersionRunByModelUID(ctx context.Context, userUID string, modelUID string, version string) (modelRun *datamodel.ModelRun, err error)
	ListModelRuns(ctx context.Context, pageSize, page int64, filter filtering.Filter, order ordering.OrderBy, requesterUID string, isOwner bool, modelUID string) (modelRuns []*datamodel.ModelRun, totalSize int64, err error)
	CreateModelRun(ctx context.Context, modelRun *datamodel.ModelRun) (*datamodel.ModelRun, error)
	UpdateModelRun(ctx context.Context, modelRun *datamodel.ModelRun) error
	ListModelRunsByRequester(ctx context.Context, params *ListModelRunsByRequesterParams) (modelTriggers []*datamodel.ModelRun, totalSize int64, err error)
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

func (r *repository) CheckPinnedUser(ctx context.Context, db *gorm.DB, table string) *gorm.DB {
	userUID := resourcex.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// If the user is pinned, we will use the primary database for querying.
	if !errors.Is(r.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table)).Err(), redis.Nil) {
		db = db.Clauses(dbresolver.Write)
	}
	return db
}

func (r *repository) PinUser(ctx context.Context, table string) {
	userUID := resourcex.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// To solve the read-after-write inconsistency problem,
	// we will direct the user to read from the primary database for a certain time frame
	// to ensure that the data is synchronized from the primary DB to the replica DB.
	_ = r.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table), time.Now(), time.Duration(config.Config.Database.Replica.ReplicationTimeFrame)*time.Second)
}

func (r *repository) listModels(ctx context.Context, where string, whereArgs []any, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {

	logger, _ := logx.GetZapLogger(ctx)

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter, "model"); err != nil {
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

	joinStr := "left join model_tag on model_tag.model_uid = model.uid left join model_version on model_version.model_uid = model.uid"

	countBuilder := db.Distinct("uid").Model(&datamodel.Model{}).Where(where, whereArgs...).Joins(joinStr)
	if uidAllowList != nil {
		countBuilder = countBuilder.Where("uid in ?", uidAllowList).Count(&totalSize)
	}

	countBuilder.Count(&totalSize)

	queryBuilder := db.Distinct().Model(&datamodel.Model{}).Joins(joinStr).Where(where, whereArgs...)
	if len(order.Fields) == 0 {
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
			return nil, 0, "", errorsx.NewPageTokenErr(err)
		}

		for _, o := range order.Fields {

			p := strcase.ToSnake(o.Path)
			if v, ok := tokens[p]; ok {
				switch p {
				case datamodel.FieldCreateTime, datamodel.FieldUpdateTime:
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

	result := queryBuilder.Preload("Tags").Preload("Versions").Find(&models)
	if result.Error != nil {
		logger.Error(result.Error.Error())
		return nil, 0, "", result.Error
	}

	if len(models) > 0 {
		lastUID := (models)[len(models)-1].UID
		lastItem := &datamodel.Model{}

		tokens := map[string]any{}

		lastItemQueryBuilder := db.Distinct().Model(&datamodel.Model{}).Joins(joinStr).Omit("profile_image").Where(where, whereArgs...)
		if uidAllowList != nil {
			lastItemQueryBuilder = lastItemQueryBuilder.Where("uid in ?", uidAllowList)
		}

		for _, field := range order.Fields {
			orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(!field.Desc)
			lastItemQueryBuilder.Order(orderString)
			switch p := strcase.ToSnake(field.Path); p {
			// todo: this is not being used?
			case "id":
				tokens[p] = (models)[len(models)-1].ID
			case datamodel.FieldCreateTime:
				tokens[p] = (models)[len(models)-1].CreateTime.Format(time.RFC3339Nano)
			case datamodel.FieldUpdateTime:
				tokens[p] = (models)[len(models)-1].UpdateTime.Format(time.RFC3339Nano)
			case datamodel.FieldLastRunTime:
				tokens[p] = (models)[len(models)-1].LastRunTime.Format(time.RFC3339Nano)
			case datamodel.FieldNumberOfRuns:
				tokens[p] = (models)[len(models)-1].NumberOfRuns
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

func (r *repository) ListModels(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy, visibility *modelpb.Model_Visibility) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {
	where := ""
	whereArgs := []any{}
	if *visibility != modelpb.Model_VISIBILITY_UNSPECIFIED {
		where = "(visibility = ?)"
		whereArgs = append(whereArgs, datamodel.ModelVisibility(*visibility))
	}
	return r.listModels(ctx,
		where,
		whereArgs,
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, order)
}

func (r *repository) ListNamespaceModels(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, order ordering.OrderBy, visibility *modelpb.Model_Visibility) (models []*datamodel.Model, totalSize int64, nextPageToken string, err error) {
	where := "(owner = ?)"
	whereArgs := []any{ownerPermalink}
	if *visibility != modelpb.Model_VISIBILITY_UNSPECIFIED {
		where = "(owner = ? AND visibility = ?)"
		whereArgs = append(whereArgs, datamodel.ModelVisibility(*visibility))
	}
	return r.listModels(ctx,
		where,
		whereArgs,
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, order)
}

func (r *repository) ListModelsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Model, int64, string, error) {
	return r.listModels(ctx, "", []any{}, pageSize, pageToken, isBasicView, filter, nil, showDeleted, ordering.OrderBy{})
}

func (r *repository) getNamespaceModel(ctx context.Context, where string, whereArgs []any, isBasicView bool, includeAvatar bool) (*datamodel.Model, error) {

	db := r.CheckPinnedUser(ctx, r.db, "model")

	var model datamodel.Model

	queryBuilder := db.Model(&datamodel.Model{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}
	if !includeAvatar {
		queryBuilder.Omit("profile_image")
	}

	if result := queryBuilder.First(&model); result.Error != nil {
		return nil, result.Error
	}

	model.Tags = []*datamodel.ModelTag{}
	tagDB := r.CheckPinnedUser(ctx, r.db, "model_tag")
	tagDBQueryBuilder := tagDB.Model(&datamodel.ModelTag{}).Where("model_uid = ?", model.UID)
	tagDBQueryBuilder.Find(&model.Tags)

	model.Versions = []*datamodel.ModelVersion{}
	versionDB := r.CheckPinnedUser(ctx, r.db, "model_version")
	versionDBQueryBuilder := versionDB.Model(&datamodel.ModelVersion{}).Where("model_uid = ?", model.UID)
	versionDBQueryBuilder.Find(&model.Versions)

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

	r.PinUser(ctx, "model")
	db := r.CheckPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).Create(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *repository) UpdateNamespaceModelByID(ctx context.Context, ownerPermalink string, id string, model *datamodel.Model) error {

	r.PinUser(ctx, "model")
	db := r.CheckPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Updates(model); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return errorsx.ErrNoDataUpdated
	}
	return nil
}

func (r *repository) UpdateNamespaceModelIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {

	r.PinUser(ctx, "model")
	db := r.CheckPinnedUser(ctx, r.db, "model")

	if result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return errorsx.ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespaceModelByID(ctx context.Context, ownerPermalink string, id string) error {

	r.PinUser(ctx, "model")
	db := r.CheckPinnedUser(ctx, r.db, "model")

	result := db.Model(&datamodel.Model{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Delete(&datamodel.Model{})

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return status.Errorf(codes.NotFound, "The model ID %s not found", id)
		}
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNoDataDeleted
	}

	return nil
}

func (r *repository) CreateModelVersion(ctx context.Context, ownerPermalink string, version *datamodel.ModelVersion) error {

	r.PinUser(ctx, "model_version")
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).Create(&version); result.Error != nil {

		var pgErr *pgconn.PgError

		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errorsx.ErrAlreadyExists
		}

		return result.Error
	}

	return nil
}

func (r *repository) UpdateModelVersionDigestByID(ctx context.Context, modelUID uuid.UUID, versionID string, digest string) error {

	r.PinUser(ctx, "model_version")
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).
		Where("(version = ? AND model_uid = ?)", versionID, modelUID).
		Update("digest", digest); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return errorsx.ErrNoDataUpdated
	}

	return nil
}

func (r *repository) DeleteModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) error {

	r.PinUser(ctx, "model_version")
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	result := db.Model(&datamodel.ModelVersion{}).
		Where("(version = ? AND model_uid = ?)", versionID, modelUID).
		Delete(&datamodel.ModelVersion{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNoDataDeleted
	}

	return nil
}

func (r *repository) DeleteModelVersionByDigest(ctx context.Context, modelUID uuid.UUID, digest string) error {

	r.PinUser(ctx, "model_version")
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	result := db.Model(&datamodel.ModelVersion{}).
		Where("(digest = ? AND model_uid = ?)", digest, modelUID).
		Delete(&datamodel.ModelVersion{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNoDataDeleted
	}

	return nil
}

func (r *repository) GetLatestModelVersionByModelUID(ctx context.Context, modelUID uuid.UUID) (version *datamodel.ModelVersion, err error) {
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	queryBuilder := db.Model(&datamodel.ModelVersion{}).Where("(model_uid = ?)", modelUID)

	if result := queryBuilder.Order("update_time DESC").First(&version); result.Error != nil {
		return nil, result.Error
	}
	return version, nil
}

func (r *repository) GetModelVersionByID(ctx context.Context, modelUID uuid.UUID, versionID string) (version *datamodel.ModelVersion, err error) {
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	queryBuilder := db.Model(&datamodel.ModelVersion{}).Where("(version = ? AND model_uid = ?)", versionID, modelUID)

	if result := queryBuilder.First(&version); result.Error != nil {
		return nil, result.Error
	}
	return version, nil
}

func (r *repository) ListModelVersionsByDigest(ctx context.Context, modelUID uuid.UUID, digest string) (versions []*datamodel.ModelVersion, err error) {
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	if result := db.Model(&datamodel.ModelVersion{}).Where("(digest = ? AND model_uid = ?)", digest, modelUID).Find(&versions); result.Error != nil {
		return []*datamodel.ModelVersion{}, status.Errorf(codes.NotFound, "The model versions belongs to model uid %v with digest %s not found", modelUID, digest)
	}
	return versions, nil
}

func (r *repository) ListModelVersions(ctx context.Context, modelUID uuid.UUID, groupDigest bool) (versions []*datamodel.ModelVersion, err error) {
	db := r.CheckPinnedUser(ctx, r.db, "model_version")

	queryBuilder := db.Model(&datamodel.ModelVersion{}).Where("model_uid", modelUID)

	if groupDigest {
		queryBuilder.Select("MIN(version) as version, model_uid, digest").Group("model_uid, digest")
	}

	if result := queryBuilder.Find(&versions); result.Error != nil {
		return []*datamodel.ModelVersion{}, status.Errorf(codes.NotFound, "The model versions belongs to model uid %v not found", modelUID)
	}
	return versions, nil
}

func (r *repository) CreateModelTags(ctx context.Context, modelUID uuid.UUID, tagNames []string) error {

	r.PinUser(ctx, "model_tag")

	db := r.CheckPinnedUser(ctx, r.db, "model_tag")

	tags := []datamodel.ModelTag{}
	for _, tagName := range tagNames {
		tag := datamodel.ModelTag{
			ModelUID:   modelUID,
			TagName:    tagName,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		tags = append(tags, tag)
	}

	if result := db.Model(&datamodel.ModelTag{}).Create(&tags); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errorsx.ErrAlreadyExists
		}
		return result.Error
	}

	return nil

}

func (r *repository) DeleteModelTags(ctx context.Context, modelUID uuid.UUID, tagNames []string) error {

	r.PinUser(ctx, "model_tag")

	db := r.CheckPinnedUser(ctx, r.db, "model_tag")

	result := db.Model(&datamodel.ModelTag{}).Where("model_uid = ? and tag_name in ?", modelUID, tagNames).Delete(&datamodel.ModelTag{})

	if result.Error != nil {

		return result.Error

	}

	if result.RowsAffected == 0 {

		return errorsx.ErrNoDataDeleted

	}

	return nil

}

func (r *repository) ListModelTags(ctx context.Context, modelUID uuid.UUID) ([]datamodel.ModelTag, error) {

	db := r.CheckPinnedUser(ctx, r.db, "model_tag")

	var tags []datamodel.ModelTag

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
		return nil, "", 0, status.Error(codes.Internal, result.Error.Error())
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
			return nil, "", 0, errorsx.NewPageTokenErr(err)
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, id)
	}

	if view != modelpb.View_VIEW_FULL {
		queryBuilder.Omit("model_spec")
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", 0, status.Error(codes.Internal, err.Error())
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
			return nil, "", 0, status.Error(codes.Internal, result.Error.Error())
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
func (r *repository) transpileFilter(filter filtering.Filter, tableName string) (*clause.Expr, error) {
	return (&Transpiler{
		filter:    filter,
		tableName: tableName,
	}).Transpile()
}

func (r *repository) GetModelRunByUID(ctx context.Context, triggerUID string) (modelTrigger *datamodel.ModelRun, err error) {
	return r.getModelRunByModelUID(
		ctx,
		"(uid = ?)",
		[]any{triggerUID},
	)
}

func (r *repository) GetLatestModelRunByModelUID(ctx context.Context, userUID string, modelUID string) (modelTrigger *datamodel.ModelRun, err error) {
	return r.getModelRunByModelUID(
		ctx,
		"(model_uid = ? AND requester_uid = ?)",
		[]any{modelUID, userUID},
	)
}

func (r *repository) GetLatestModelVersionRunByModelUID(ctx context.Context, userUID string, modelUID string, version string) (modelTrigger *datamodel.ModelRun, err error) {
	return r.getModelRunByModelUID(
		ctx,
		"(model_uid = ? AND requester_uid = ? AND model_version = ?)",
		[]any{modelUID, userUID, version},
	)
}

const tableModelRun = "model_run"

func (r *repository) getModelRunByModelUID(ctx context.Context, where string, whereArgs []any) (modelTrigger *datamodel.ModelRun, err error) {

	db := r.CheckPinnedUser(ctx, r.db, tableModelRun)

	var trigger datamodel.ModelRun

	queryBuilder := db.Model(&datamodel.ModelRun{}).Where(where, whereArgs...)
	if result := queryBuilder.First(&trigger); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "The model trigger not found")
		}
		return nil, status.Error(codes.Internal, result.Error.Error())
	}

	return &trigger, nil
}

func (r *repository) ListModelRuns(ctx context.Context, pageSize, page int64, filter filtering.Filter, order ordering.OrderBy,
	requesterUID string, isOwner bool, modelUID string) (modelRuns []*datamodel.ModelRun, totalSize int64, err error) {

	logger, _ := logx.GetZapLogger(ctx)

	db := r.CheckPinnedUser(ctx, r.db, tableModelRun)

	whereConditions := []string{"model_uid = ?"}
	whereArgs := []any{modelUID}

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter, tableModelRun); err != nil {
		return nil, 0, err
	}
	if expr != nil {
		whereConditions = append(whereConditions, "(?)")
		whereArgs = append(whereArgs, expr)
	}

	if !isOwner {
		whereConditions = append(whereConditions, "requester_uid = ?")
		whereArgs = append(whereArgs, requesterUID)
	}

	var where string
	if len(whereConditions) > 0 {
		where = strings.Join(whereConditions, " and ")
	}

	if err = db.Model(&datamodel.ModelRun{}).Where(where, whereArgs...).Count(&totalSize).Error; err != nil {
		logger.Error("failed in count model run total size", zap.Error(err))
		return nil, 0, err
	}

	queryBuilder := db.Where(where, whereArgs...)
	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "create_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}

	if err = queryBuilder.Limit(int(pageSize)).Offset(int(pageSize * page)).Find(&modelRuns).Error; err != nil {
		logger.Error("failed in querying model runs", zap.Error(err))
		return nil, 0, err
	}

	return modelRuns, totalSize, nil
}

func (r *repository) CreateModelRun(ctx context.Context, modelRun *datamodel.ModelRun) (*datamodel.ModelRun, error) {

	r.PinUser(ctx, "model")
	db := r.CheckPinnedUser(ctx, r.db, "model")

	result := db.Model(&datamodel.Model{}).
		Where("uid = ?", modelRun.ModelUID).
		UpdateColumns(map[string]any{
			"last_run_time":  time.Now(),
			"number_of_runs": gorm.Expr("number_of_runs + 1"),
		})
	if result.Error != nil {
		return nil, result.Error
	}

	r.PinUser(ctx, tableModelRun)
	db = r.CheckPinnedUser(ctx, r.db, tableModelRun)

	if err := db.Create(modelRun).Error; err != nil {
		return nil, err
	}
	return modelRun, nil
}

func (r *repository) UpdateModelRun(ctx context.Context, modelRun *datamodel.ModelRun) error {

	r.PinUser(ctx, tableModelRun)
	return r.CheckPinnedUser(ctx, r.db, tableModelRun).Model(&datamodel.ModelRun{}).
		Where(&datamodel.ModelRun{BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: modelRun.UID}}).
		Updates(&modelRun).Error
}

// ListModelRunsByRequesterParams is the parameters for listing model runs by requester
type ListModelRunsByRequesterParams struct {
	PageSize         int64
	Page             int64
	Filter           filtering.Filter
	Order            ordering.OrderBy
	RequesterUID     string
	StartedTimeBegin time.Time
	StartedTimeEnd   time.Time
}

// ListModelRunsByRequester lists model runs by requester
func (r *repository) ListModelRunsByRequester(ctx context.Context, params *ListModelRunsByRequesterParams) ([]*datamodel.ModelRun, int64, error) {

	logger, _ := logx.GetZapLogger(ctx)

	var modelRuns []*datamodel.ModelRun
	var totalSize int64
	var err error

	db := r.CheckPinnedUser(ctx, r.db, tableModelRun)

	whereConditions := []string{"requester_uid = ? and create_time >= ? and create_time <= ?"}
	whereArgs := []any{params.RequesterUID, params.StartedTimeBegin, params.StartedTimeEnd}

	var expr *clause.Expr
	if expr, err = r.transpileFilter(params.Filter, tableModelRun); err != nil {
		return nil, 0, err
	}
	if expr != nil {
		whereConditions = append(whereConditions, "(?)")
		whereArgs = append(whereArgs, expr)
	}

	var where string
	if len(whereConditions) > 0 {
		where = strings.Join(whereConditions, " and ")
	}

	if err = db.Model(&datamodel.ModelRun{}).Where(where, whereArgs...).Count(&totalSize).Error; err != nil {
		logger.Error("failed in count model run total size", zap.Error(err))
		return nil, 0, err
	}

	queryBuilder := db.Preload(clause.Associations).Where(where, whereArgs...)
	order := params.Order
	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "create_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}

	if err = queryBuilder.Limit(int(params.PageSize)).Offset(int(params.PageSize * params.Page)).Find(&modelRuns).Error; err != nil {
		logger.Error("failed in querying model runs", zap.Error(err))
		return nil, 0, err
	}

	return modelRuns, totalSize, nil
}
