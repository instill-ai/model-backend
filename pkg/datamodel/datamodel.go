package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ModelState modelPB.Model_State
type ModelVisibility modelPB.Model_Visibility
type ModelTask commonPB.Task

type BaseStatic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;"`
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamic) BeforeCreate(db *gorm.DB) error {
	recordUUID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("UID", recordUUID)
	return nil
}

type ModelDefinition struct {
	BaseStatic

	// ModelDefinition id
	ID string `json:"id,omitempty"`

	// ModelDefinition title
	Title string `json:"title,omitempty"`

	// ModelDefinition documentation_url
	DocumentationURL string `json:"documentation_url,omitempty"`

	// ModelDefinition icon
	Icon string `json:"icon,omitempty"`

	// ModelDefinition model spec
	ModelSpec datatypes.JSON `json:"model_spec,omitempty"`

	ReleaseStage ReleaseStage `sql:"type:valid_release_stage"`
}

// Model
type Model struct {
	BaseDynamic

	// Model id
	ID string `json:"id,omitempty"`

	// Model description
	Description sql.NullString

	// Model definition
	ModelDefinitionUID uuid.UUID `gorm:"model_definition_uid,omitempty"`

	// Model definition configuration
	Configuration datatypes.JSON `json:"configuration,omitempty"`

	// Model visibility
	Visibility ModelVisibility `json:"visibility,omitempty"`

	// Model owner
	Owner string `json:"owner,omitempty"`

	// Model task
	Task ModelTask `json:"task,omitempty"`

	// Model state
	State ModelState `json:"state,omitempty"`
}

type ModelInferResult struct {
	BaseDynamic

	// Inference id: `model id.{datetime}.infer` created by temporal
	ID string `json:"id,omitempty"`

	// Inference result
	Result datatypes.JSON `json:"result,omitempty"`

	// Model uid
	ModelUID uuid.UUID `json:"model_uid,omitempty"`
}

// Model configuration
type GitHubModelConfiguration struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	HTMLURL    string `json:"html_url,omitempty"`
}

type ArtiVCModelConfiguration struct {
	URL        string         `json:"url,omitempty"`
	Tag        string         `json:"tag,omitempty"`
	Credential datatypes.JSON `json:"credential,omitempty"`
}

type HuggingFaceModelConfiguration struct {
	RepoID  string `json:"repo_id,omitempty"`
	Tag     string `json:"tag,omitempty"`
	HTMLURL string `json:"html_url,omitempty"`
}

type LocalModelConfiguration struct {
	Content string `json:"content,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type ContainerizedModelConfiguration struct {
	Task string `json:"task,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

type ListModelQuery struct {
	Owner string
}

func (s *ModelState) Scan(value any) error {
	*s = ModelState(modelPB.Model_State_value[value.(string)])
	return nil
}

func (s ModelTask) Value() (driver.Value, error) {
	return commonPB.Task(s).String(), nil
}

func (s *ModelTask) Scan(value any) error {
	*s = ModelTask(commonPB.Task_value[value.(string)])
	return nil
}

func (s ModelState) Value() (driver.Value, error) {
	return modelPB.Model_State(s).String(), nil
}

func (v *ModelVisibility) Scan(value any) error {
	*v = ModelVisibility(modelPB.Model_Visibility_value[value.(string)])
	return nil
}

func (v ModelVisibility) Value() (driver.Value, error) {
	return modelPB.Model_Visibility(v).String(), nil
}

// ReleaseStage is an alias type for Protobuf enum ReleaseStage
type ReleaseStage modelPB.ReleaseStage

// Scan function for custom GORM type ReleaseStage
func (r *ReleaseStage) Scan(value any) error {
	*r = ReleaseStage(modelPB.ReleaseStage_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ReleaseStage) Value() (driver.Value, error) {
	return modelPB.ReleaseStage(r).String(), nil
}
