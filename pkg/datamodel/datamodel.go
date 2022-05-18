package datamodel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

type ModelInstanceState modelPB.ModelInstance_State
type ModelVisibility modelPB.Model_Visibility
type ModelInstanceTask modelPB.ModelInstance_Task

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
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("UID", uuid)
	return nil
}

type ModelDefinition struct {
	BaseStatic

	// ModelDefinition id
	ID string `json:"id,omitempty"`

	// ModelDefinition title
	Title string `json:"title,omitempty"`

	// ModelDefinition documentation_url
	DocumentationUrl string `json:"documentation_url,omitempty"`

	// ModelDefinition icon
	Icon string `json:"icon,omitempty"`

	// ModelDefinition model spec
	ModelSpec datatypes.JSON `json:"model_spec,omitempty"`

	// ModelDefinition model instance spec
	ModelInstanceSpec datatypes.JSON `json:"model_instance_spec,omitempty"`
}

// Model combines several Triton model. It includes ensemble model.
type Model struct {
	BaseDynamic

	// Model id
	ID string `json:"id,omitempty"`

	// Model description
	Description string `json:"description,omitempty"`

	// Model definition
	ModelDefinition string `json:"model_definition,omitempty"`

	// Model definition configuration
	Configuration datatypes.JSON `json:"configuration,omitempty"`

	// Model visibility
	Visibility ModelVisibility `json:"visibility,omitempty"`

	// Model owner
	Owner string `json:"owner,omitempty"`

	// Not stored in DB, only used for processing
	Instances []ModelInstance `gorm:"foreignKey:ModelUID;references:UID;constraint:OnDelete:CASCADE;"`
}

// Triton model
type TritonModel struct {
	BaseDynamic

	// Triton Model name
	Name string `json:"name,omitempty"`

	// Triton Model version
	Version int `json:"version,omitempty"`

	// Triton Model status
	State ModelInstanceState `json:"state,omitempty"`

	// Model triton platform, only store ensemble model to make inferencing
	Platform string `json:"platform,omitempty"`

	// Model Instance uid
	ModelInstanceUID uuid.UUID `json:"model_instance_uid,omitempty"`
}

type ModelInstance struct {
	BaseDynamic

	// Model Instance id
	ID string `json:"id,omitempty"`

	// Model instance status
	State ModelInstanceState `sql:"type:valid_state"`

	// Model instance task
	Task ModelInstanceTask `json:"task,omitempty"`

	// Model definition
	ModelDefinition string `json:"model_definition,omitempty"`

	// Model instance configuration
	Configuration datatypes.JSON `json:"configuration,omitempty"`

	// Model id
	ModelUID uuid.UUID `json:"model_uid,omitempty"`

	// Not stored in DB, only used for processing
	TritonModels []TritonModel `gorm:"foreignKey:ModelInstanceUID;references:UID;constraint:OnDelete:CASCADE;"`
}

// Model configuration
type GitHubModelConfiguration struct {
	Repository string `json:"repository,omitempty"`
	HtmlUrl    string `json:"html_url,omitempty"`
}

// Model Instance configuration
type GitHubModelInstanceConfiguration struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	HtmlUrl    string `json:"html_url,omitempty"`
}

type LocalModelConfiguration struct {
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
}

type ListModelQuery struct {
	Owner string
}

func (s *ModelInstanceState) Scan(value interface{}) error {
	*s = ModelInstanceState(modelPB.ModelInstance_State_value[value.(string)])
	return nil
}

func (s ModelInstanceTask) Value() (driver.Value, error) {
	return modelPB.ModelInstance_Task(s).String(), nil
}

func (s *ModelInstanceTask) Scan(value interface{}) error {
	*s = ModelInstanceTask(modelPB.ModelInstance_Task_value[value.(string)])
	return nil
}

func (s ModelInstanceState) Value() (driver.Value, error) {
	return modelPB.ModelInstance_State(s).String(), nil
}

func (v *ModelVisibility) Scan(value interface{}) error {
	*v = ModelVisibility(modelPB.Model_Visibility_value[value.(string)])
	return nil
}

func (v ModelVisibility) Value() (driver.Value, error) {
	return modelPB.Model_Visibility(v).String(), nil
}

func (r *ModelInstance) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal ModelInstance value:", value))
	}

	if err := json.Unmarshal(bytes, &r); err != nil {
		return err
	}
	return nil
}

func (r *ModelInstance) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}
