package datamodel

import (
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	"github.com/shopspring/decimal"
	"gopkg.in/guregu/null.v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TriggerStatus modelpb.ModelTrigger_TriggerStatus
type TriggerSource modelpb.ModelTrigger_TriggerSource

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
	UID               uuid.UUID `gorm:"primary_key"`
	ModelUID          uuid.UUID
	TriggerUID        uuid.UUID
	ModelVersion      string
	ModelTask         ModelTask
	ModelTags         datatypes.JSON
	Status            TriggerStatus
	Visibility        ModelVisibility
	Source            TriggerSource
	StartTime         time.Time
	TotalDuration     null.Int
	EndTime           null.Time
	RequesterUID      uuid.UUID
	InputReferenceID  string
	OutputReferenceID null.String
	Credits           decimal.Decimal
	Error             null.String
	CreateTime        time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime        time.Time `gorm:"autoUpdateTime:nano"`
}

func (l *ModelTrigger) BeforeCreate(db *gorm.DB) error {
	recordUUID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	l.UID = recordUUID
	return nil
}
