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
)

// BaseStatic contains common columns for all tables with static UUID as primary key
type BaseStatic struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamic) BeforeCreate(db *gorm.DB) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("ID", uuid)
	return nil
}

// Model combines several Triton model. It includes ensemble model.
type Model struct {
	BaseDynamic

	// Model name
	Name string `json:"name,omitempty"`

	// workspace name where model belong to
	Namespace string `json:"namespace,omitempty"`

	// Model visibility
	Visibility string `json:"visibility,omitempty"`

	// Model description
	Description string `json:"description,omitempty"`

	// Model source
	Source string `json:"source,omitempty"`

	// Model configuration
	Config datatypes.JSON `gorm:"config:jsonb"`

	// Model Owner
	Owner datatypes.JSON `gorm:"owner:jsonb"`

	// Not stored in DB, only used for processing
	FullName     string
	TritonModels []TritonModel `gorm:"foreignKey:ModelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Instances    []Instance    `gorm:"foreignKey:ModelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Triton model
type TritonModel struct {
	BaseDynamic

	// Triton Model name
	Name string `json:"name,omitempty"`

	// Triton Model version
	Version int `json:"version,omitempty"`

	// Triton Model status
	Status string `json:"status,omitempty"`

	// Model ID
	ModelID uuid.UUID `json:"model_id,omitempty"`

	// Model Instance Name
	ModelInstance string `json:"model_instance,omitempty"`

	// Model triton platform, only store ensemble model to make inferencing
	Platform string `json:"platform,omitempty"`
}

type Instance struct {
	BaseDynamic

	// Model ID
	ModelID uuid.UUID `json:"model_id,omitempty" gorm:"column:model_id"`

	// Model instance name
	Name string `json:"name,omitempty"`

	// Model instance task
	Task uint `json:"task,omitempty"`

	// Model version status
	Status ValidStatus `sql:"type:valid_status"`

	// Model instance configuration
	Config datatypes.JSON `gorm:"configuration:jsonb"`

	// Output only, not store in DB
	Source            string `json:"source,omitempty"`
	ModelDefinitionId string `json:"model_definition_id,omitempty"`
}

// Model configuration
type ModelConfiguration struct {
	Repo    string `json:"repo,omitempty"`
	HtmlUrl string `json:"html_url,omitempty"`
}

// Model Instance configuration
type InstanceConfiguration struct {
	Repo    string `json:"repo,omitempty"`
	Tag     string `json:"tag,omitempty"`
	HtmlUrl string `json:"html_url,omitempty"`
}

type Owner struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Type     string `json:"type,omitempty"`
}

type ListModelQuery struct {
	Namespace string
}

type ValidStatus string

const (
	StatusUnspecified ValidStatus = "STATUS_UNSPECIFIED"
	StatusOffline     ValidStatus = "STATUS_OFFLINE"
	StatusOnline      ValidStatus = "STATUS_ONLINE"
	StatusError       ValidStatus = "STATUS_ERROR"
)

func (p *ValidStatus) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidStatus(v)
	case []byte:
		*p = ValidStatus(v)
	default:
		return errors.New("Incompatible type for ValidStatus")
	}
	return nil
}

func (p ValidStatus) Value() (driver.Value, error) {
	return string(p), nil
}

func (r *Instance) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal TritonModel value:", value))
	}

	if err := json.Unmarshal(bytes, &r); err != nil {
		return err
	}

	return nil
}

func (r *Instance) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}
