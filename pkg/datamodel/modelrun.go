package datamodel

import (
	"database/sql/driver"

	"github.com/gofrs/uuid"
	"gopkg.in/guregu/null.v4"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
)

// for saving the protobuf types as string values
type (
	RunStatus runpb.RunStatus
	RunSource runpb.RunSource
)

func (v *RunStatus) Scan(value any) error {
	*v = RunStatus(runpb.RunStatus_value[value.(string)])
	return nil
}

func (v RunStatus) Value() (driver.Value, error) {
	return runpb.RunStatus(v).String(), nil
}

func (v *RunSource) Scan(value any) error {
	*v = RunSource(runpb.RunSource_value[value.(string)])
	return nil
}

func (v RunSource) Value() (driver.Value, error) {
	return runpb.RunSource(v).String(), nil
}

type ModelRun struct {
	BaseStaticHardDelete
	ModelUID          uuid.UUID
	ModelVersion      string
	Status            RunStatus
	Source            RunSource
	TotalDuration     null.Int
	EndTime           null.Time
	RequesterUID      uuid.UUID
	RunnerUID         uuid.UUID
	InputReferenceID  string
	OutputReferenceID null.String
	Error             null.String
}

func (*ModelRun) TableName() string {
	return "model_trigger"
}
