package datamodel

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Model combines several Triton model. It includes ensemble model.
type Model struct {

	// Model unique ID
	Id uint64 `json:"id,omitempty"`

	// Model name
	Name string `json:"name,omitempty"`

	// workspace name where model belong to
	Namespace string `json:"namespace,omitempty"`

	Task uint64 `json:"task,omitempty"`

	// Not stored inn DB, only used for processing
	FullName     string
	TritonModels []TModel
	Versions     []Version
}

// Triton model
type TModel struct {

	// Triton Model unique ID
	Id uint64 `json:"id,omitempty"`

	// Triton Model name
	Name string `json:"name,omitempty"`

	// Triton Model version
	Version int `json:"version,omitempty"`

	// Triton Model status
	Status string `json:"status,omitempty"`

	// Model ID
	ModelId uint64 `json:"model_id,omitempty"`

	ModelVersion uint64 `json:"model_version,omitempty"`

	Platform string `json:"platform,omitempty"`
}

type Version struct {
	// Model ID
	ModelId uint64 `json:"model_id,omitempty"`

	// Model version
	Version uint64 `json:"version,omitempty"`

	// Model description
	Description string `json:"description,omitempty"`

	// Model version created date time
	CreatedAt time.Time `gorm:"type:timestamp"`

	// Model version updated date time
	UpdatedAt time.Time `gorm:"type:timestamp"`

	// Model version status
	Status string `json:"status,omitempty"`

	// Model version metadata
	Metadata JSONB `gorm:"type:jsonb"`
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

type ListModelQuery struct {
	Namespace string
}
