package datamodel

import (
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	"gopkg.in/guregu/null.v4"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

// for saving the protobuf types as string values
type (
	TriggerStatus modelpb.ModelRun_RunStatus
	TriggerSource modelpb.ModelRun_RunSource
)

func (v *TriggerStatus) Scan(value any) error {
	*v = TriggerStatus(modelpb.ModelRun_RunStatus_value[value.(string)])
	return nil
}

func (v TriggerStatus) Value() (driver.Value, error) {
	return modelpb.ModelRun_RunStatus(v).String(), nil
}

func (v *TriggerSource) Scan(value any) error {
	*v = TriggerSource(modelpb.ModelRun_RunSource_value[value.(string)])
	return nil
}

func (v TriggerSource) Value() (driver.Value, error) {
	return modelpb.ModelRun_RunSource(v).String(), nil
}

type ModelTrigger struct {
	UID               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ModelUID          uuid.UUID
	ModelVersion      string
	Status            TriggerStatus
	Source            TriggerSource
	TotalDuration     null.Int
	EndTime           null.Time
	RequesterUID      uuid.UUID
	InputReferenceID  string
	OutputReferenceID null.String
	Error             null.String
	CreateTime        time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime        time.Time `gorm:"autoUpdateTime:nano"`
}
