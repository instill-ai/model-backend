package datamodel

import (
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	"gopkg.in/guregu/null.v4"
)

// for saving the protobuf types as string values
type (
	TriggerStatus modelpb.ModelTrigger_TriggerStatus
	TriggerSource modelpb.ModelTrigger_TriggerSource
)

func (v *TriggerStatus) Scan(value any) error {
	*v = TriggerStatus(modelpb.ModelTrigger_TriggerStatus_value[value.(string)])
	return nil
}

func (v TriggerStatus) Value() (driver.Value, error) {
	return modelpb.ModelTrigger_TriggerStatus(v).String(), nil
}

func (v *TriggerSource) Scan(value any) error {
	*v = TriggerSource(modelpb.ModelTrigger_TriggerSource_value[value.(string)])
	return nil
}

func (v TriggerSource) Value() (driver.Value, error) {
	return modelpb.ModelTrigger_TriggerSource(v).String(), nil
}

type ModelTrigger struct {
	UID               uuid.UUID `gorm:"primary_key;<-:false"`
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
