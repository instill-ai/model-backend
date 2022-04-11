package datamodel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Model combines several Triton model. It includes ensemble model.
type Model struct {
	gorm.Model

	// Model name
	Name string `json:"name,omitempty"`

	// workspace name where model belong to
	Namespace string `json:"namespace,omitempty"`

	Task uint `json:"task,omitempty"`

	// Not stored in DB, only used for processing
	FullName     string
	TritonModels []TritonModel `gorm:"foreignKey:ModelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Versions     []Version     `gorm:"foreignKey:ModelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type GitRef struct {
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Commit string `json:"commit,omitempty"`
}
type GitHub struct {
	// Model github repository URL
	RepoUrl string `json:"repo_url,omitempty"`
	GitRef  GitRef `json:"git_ref,omitempty"`
}

// Triton model
type TritonModel struct {
	gorm.Model

	// Triton Model name
	Name string `json:"name,omitempty"`

	// Triton Model version
	Version int `json:"version,omitempty"`

	// Triton Model status
	Status string `json:"status,omitempty"`

	// Model ID
	ModelID uint `json:"model_id,omitempty"`

	ModelVersion uint `json:"model_version,omitempty"`

	Platform string `json:"platform,omitempty"`
}

type Version struct {
	gorm.Model

	// Model ID
	ModelID uint `json:"model_id,omitempty" gorm:"column:model_id"`

	// Model version
	Version uint `json:"version,omitempty"`

	// Model description
	Description string `json:"description,omitempty"`

	// Model version status
	Status ValidStatus `sql:"type:valid_status"`

	// Model version metadata
	Metadata JSONB `gorm:"type:jsonb"`

	// GitHub information corresponding to a model version
	// It will empty if model is created by local file
	Github GitHub `gorm:"type:jsonb"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	valueString, err := json.Marshal(j)
	return string(valueString), err
}

func (j *JSONB) Scan(value interface{}) error {
	if err := json.Unmarshal(value.([]byte), &j); err != nil {
		return err
	}
	return nil
}

func (j GitHub) Value() (driver.Value, error) {
	valueString, err := json.Marshal(j)
	return string(valueString), err
}

func (j *GitHub) Scan(value interface{}) error {
	if err := json.Unmarshal(value.([]byte), &j); err != nil {
		return err
	}
	return nil
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

func (r *Version) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal TritonModel value:", value))
	}

	if err := json.Unmarshal(bytes, &r); err != nil {
		return err
	}

	return nil
}

func (r *Version) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}
