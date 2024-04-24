package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ModelVisibility modelPB.Model_Visibility
type ModelTask commonPB.Task
type UserType mgmtPB.OwnerType
type Mode mgmtPB.Mode
type Status mgmtPB.Status

type BaseStatic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;"`
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BaseStaticHardDelete contains common columns for all tables with static UUID as primary key
type BaseStaticHardDelete struct {
	UID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreateTime time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time `gorm:"autoUpdateTime:nano"`
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
	ModelSpec    datatypes.JSON `json:"model_spec,omitempty"`
	ReleaseStage ReleaseStage   `sql:"type:valid_release_stage"`
}

// Model
type Model struct {
	BaseDynamic
	ID                 string
	Description        sql.NullString
	ModelDefinitionUID uuid.UUID
	Configuration      datatypes.JSON `gorm:"type:jsonb"`
	Visibility         ModelVisibility
	Owner              string
	Task               ModelTask
	Region             string
	Hardware           string
	Readme             string
	SourceURL          string
	DocumentationURL   string
	License            string
}

// Model version
type ModelVersion struct {
	BaseDynamic
	Name     string
	Version  string
	Digest   string
	ModelUID uuid.UUID
}

type ModelPrediction struct {
	BaseStaticHardDelete
	OwnerUID            uuid.UUID      `json:"owner_uid,omitempty"`
	OwnerType           UserType       `json:"owner_type,omitempty"`
	UserUID             uuid.UUID      `json:"user_uid,omitempty"`
	UserType            UserType       `json:"user_type,omitempty"`
	Mode                Mode           `json:"mode,omitempty"`
	ModelDefinitionUID  uuid.UUID      `json:"model_definition_uid,omitempty"`
	TriggerTime         time.Time      `json:"trigger_time,omitempty"`
	ComputeTimeDuration float64        `json:"compute_time_duration,omitempty"`
	ModelTask           ModelTask      `json:"model_task,omitempty"`
	Status              Status         `json:"status,omitempty"`
	Input               datatypes.JSON `json:"input,omitempty"`
	Output              datatypes.JSON `json:"output,omitempty"`
	ModelUID            uuid.UUID      `json:"model_uid,omitempty"`
	ModelVersionUID     uuid.UUID      `json:"model_version,omitempty"`
}

type ContainerizedModelConfiguration struct {
	Task string `json:"task,omitempty"`
}

func (s ModelTask) Value() (driver.Value, error) {
	return commonPB.Task(s).String(), nil
}

func (s *ModelTask) Scan(value any) error {
	*s = ModelTask(commonPB.Task_value[value.(string)])
	return nil
}

func (v *ModelVisibility) Scan(value any) error {
	*v = ModelVisibility(modelPB.Model_Visibility_value[value.(string)])
	return nil
}

func (v ModelVisibility) Value() (driver.Value, error) {
	return modelPB.Model_Visibility(v).String(), nil
}

func (v *UserType) Scan(value any) error {
	*v = UserType(mgmtPB.OwnerType_value[value.(string)])
	return nil
}

func (v UserType) Value() (driver.Value, error) {
	return mgmtPB.OwnerType(v).String(), nil
}

func (v *Mode) Scan(value any) error {
	*v = Mode(mgmtPB.Mode_value[value.(string)])
	return nil
}

func (v Mode) Value() (driver.Value, error) {
	return mgmtPB.Mode(v).String(), nil
}

func (v *Status) Scan(value any) error {
	*v = Status(mgmtPB.Status_value[value.(string)])
	return nil
}

func (v Status) Value() (driver.Value, error) {
	return mgmtPB.Status(v).String(), nil
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
