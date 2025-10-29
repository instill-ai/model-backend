package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	// RepositoryTagTableName is the table name for repository tags
	RepositoryTagTableName = "repository_tag"
)

type repositoryTag struct {
	Name       string `gorm:"primaryKey"`
	Digest     string
	UpdateTime time.Time `gorm:"autoUpdateTime:nano"`
}

// TableName overrides the default table name for GORM
func (repositoryTag) TableName() string {
	return RepositoryTagTableName
}

func repositoryTagName(repo, id string) string {
	// In the database, the tag name is the primary key. It is compacted to
	// <repository>:tag to improve the efficiency of the queries.
	return fmt.Sprintf("%s:%s", repo, id)
}

// GetRepositoryTag fetches the tag information from the repository_tag table.
// The name param is the resource name of the tag, e.g.
// `repositories/admin/hello-world/tags/0.1.1-beta`.
func (r *repository) GetRepositoryTag(_ context.Context, name utils.RepositoryTagName) (*datamodel.Tag, error) {
	repo, tagID, err := name.ExtractRepositoryAndID()
	if err != nil {
		return nil, err
	}

	record := new(repositoryTag)
	if result := r.db.Model(record).
		Where("name = ?", repositoryTagName(repo, tagID)).
		First(record); result.Error != nil {

		if result.Error == gorm.ErrRecordNotFound {
			return nil, errorsx.ErrNotFound
		}

		return nil, result.Error
	}

	return &datamodel.Tag{
		Name:       string(name),
		ID:         tagID,
		Digest:     record.Digest,
		UpdateTime: record.UpdateTime,
	}, nil
}

// UpsertRepositoryTag stores the provided tag information in the database. The
// update timestamp will be generated on insertion.
func (r *repository) UpsertRepositoryTag(_ context.Context, tag *datamodel.Tag) (*datamodel.Tag, error) {
	repo, tagID, err := utils.RepositoryTagName(tag.Name).ExtractRepositoryAndID()
	if err != nil {
		return nil, err
	}

	record := &repositoryTag{
		Name:   repositoryTagName(repo, tagID),
		Digest: tag.Digest,
	}

	updateOnConflict := clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"digest"}),
	}
	if result := r.db.Clauses(updateOnConflict).Create(record); result.Error != nil {
		return nil, result.Error
	}

	return &datamodel.Tag{
		Name:       tag.Name,
		ID:         tag.ID,
		Digest:     record.Digest,
		UpdateTime: record.UpdateTime,
	}, nil
}

// DeleteRepositoryTag delete the tag information from the repository_tag table.
// The name param is the resource name of the tag, e.g.
// `repositories/admin/hello-world/tags/0.1.1-beta`.
func (r *repository) DeleteRepositoryTag(_ context.Context, digest string) error {
	record := new(repositoryTag)
	if result := r.db.Model(record).
		Where("digest = ?", digest).
		Delete(record); result.Error != nil {

		if result.Error == gorm.ErrRecordNotFound {
			return errorsx.ErrNotFound
		}

		return result.Error
	}

	return nil
}
