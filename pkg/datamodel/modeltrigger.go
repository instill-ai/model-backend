package datamodel

import (
	"database/sql/driver"

	"github.com/gofrs/uuid"
	"gopkg.in/guregu/null.v4"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
)

// for saving the protobuf types as string values
type (
	TriggerStatus runpb.RunStatus
	TriggerSource runpb.RunSource
)

func (v *TriggerStatus) Scan(value any) error {
	*v = TriggerStatus(runpb.RunStatus_value[value.(string)])
	return nil
}

func (v TriggerStatus) Value() (driver.Value, error) {
	return runpb.RunStatus(v).String(), nil
}

func (v *TriggerSource) Scan(value any) error {
	*v = TriggerSource(runpb.RunSource_value[value.(string)])
	return nil
}

func (v TriggerSource) Value() (driver.Value, error) {
	return runpb.RunSource(v).String(), nil
}

type ModelTrigger struct {
	BaseStaticHardDelete
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
}
