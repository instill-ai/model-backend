package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ModelVisibility modelpb.Model_Visibility
type ModelTask commonpb.Task
type UserType mgmtpb.OwnerType
type Mode mgmtpb.Mode
type Status mgmtpb.Status

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
	Readme             sql.NullString
	SourceURL          sql.NullString
	DocumentationURL   sql.NullString
	License            sql.NullString
	ProfileImage       sql.NullString
	Tags               []*ModelTag
	Versions           []*ModelVersion
	NamespaceID        string `gorm:"type:namespace_id"`
	NamespaceType      string `gorm:"type:namespace_type"`

	// Note:
	// We store the NumberOfRuns and LastRunTime in this table
	// to make it easier to sort the models. We should develop an approach to
	// sync the data between InfluxDB and here.
	LastRunTime  time.Time
	NumberOfRuns int
}

// IsPublic returns the visibility of the model.
func (m *Model) IsPublic() bool {
	return m.Visibility == ModelVisibility(modelpb.Model_VISIBILITY_PUBLIC)
}

// OwnerUID returns the UID of the model owner.
func (m *Model) OwnerUID() uuid.UUID {
	return uuid.FromStringOrNil(strings.Split(m.Owner, "/")[1])
}

func (m *Model) TagNames() []string {
	tags := make([]string, len(m.Tags))
	for i, t := range m.Tags {
		tags[i] = t.TagName
	}
	return tags
}

func (m *Model) VersionNames() []string {
	versions := make([]string, len(m.Versions))
	for i, v := range m.Versions {
		versions[i] = v.Version
	}
	return versions
}

// Model version
// Name: resource name
// Version: version name
type ModelVersion struct {
	ModelUID   uuid.UUID
	Name       string
	Version    string
	Digest     string
	CreateTime time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time `gorm:"autoUpdateTime:nano"`
}

type ModelTag struct {
	ModelUID   uuid.UUID
	TagName    string
	CreateTime time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time `gorm:"autoUpdateTime:nano"`
}

type ContainerizedModelConfiguration struct {
}

func (s ModelTask) Value() (driver.Value, error) {
	return commonpb.Task(s).String(), nil
}

func (s *ModelTask) Scan(value any) error {
	*s = ModelTask(commonpb.Task_value[value.(string)])
	return nil
}

func (v *ModelVisibility) Scan(value any) error {
	*v = ModelVisibility(modelpb.Model_Visibility_value[value.(string)])
	return nil
}

func (v ModelVisibility) Value() (driver.Value, error) {
	return modelpb.Model_Visibility(v).String(), nil
}

func (v *UserType) Scan(value any) error {
	*v = UserType(mgmtpb.OwnerType_value[value.(string)])
	return nil
}

func (v UserType) Value() (driver.Value, error) {
	return mgmtpb.OwnerType(v).String(), nil
}

func (v *Mode) Scan(value any) error {
	*v = Mode(mgmtpb.Mode_value[value.(string)])
	return nil
}

func (v Mode) Value() (driver.Value, error) {
	return mgmtpb.Mode(v).String(), nil
}

func (v *Status) Scan(value any) error {
	*v = Status(mgmtpb.Status_value[value.(string)])
	return nil
}

func (v Status) Value() (driver.Value, error) {
	return mgmtpb.Status(v).String(), nil
}

// ReleaseStage is an alias type for Protobuf enum ReleaseStage
type ReleaseStage modelpb.ReleaseStage

// Scan function for custom GORM type ReleaseStage
func (r *ReleaseStage) Scan(value any) error {
	*r = ReleaseStage(modelpb.ReleaseStage_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ReleaseStage) Value() (driver.Value, error) {
	return modelpb.ReleaseStage(r).String(), nil
}

const (
	FieldCreateTime   = "create_time"
	FieldUpdateTime   = "update_time"
	FieldLastRunTime  = "last_run_time"
	FieldNumberOfRuns = "number_of_runs"
)
