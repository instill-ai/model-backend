package utils

import (
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/santhosh-tekuri/jsonschema/v5"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
)

// Resource prefix constants for model-backend AIP-compliant IDs
const (
	PrefixModel           = "mod"
	PrefixModelDefinition = "mdf"
	PrefixModelVersion    = "ver"
	PrefixModelTag        = "mtg"
)

type ModelMeta struct {
	Tags []string
	Task string
}

type ModelSpec struct {
	ModelSpecSchema          *jsonschema.Schema `json:"model_schema,omitempty"`
	ModelConfigurationSchema *jsonschema.Schema `json:"configuration_schema,omitempty"`
}

// TODO: properly support batch inference
func DoSupportBatch() (bool, error) {
	return true, nil
}

const modelMeasurement = "model.trigger.v1"

type UsageMetricData struct {
	OwnerUID            string
	OwnerType           mgmtpb.OwnerType
	UserUID             string
	UserType            mgmtpb.OwnerType
	RequesterUID        string
	ModelID             string
	ModelUID            string
	Version             string
	Status              mgmtpb.Status
	Mode                mgmtpb.Mode
	TriggerUID          string
	ModelDefinitionUID  string
	TriggerTime         string
	ComputeTimeDuration float64
	ModelTask           commonpb.Task
}

// NewModelDataPoint transforms the information of a model trigger into
// an InfluxDB datapoint.
func NewModelDataPoint(data *UsageMetricData) *write.Point {
	// The tags contain metadata, i.e. information we might filter or group by.
	tags := map[string]string{
		"status":        data.Status.String(),
		"owner_uid":     data.OwnerUID,
		"owner_type":    data.OwnerType.String(),
		"user_uid":      data.UserUID,
		"user_type":     data.UserType.String(),
		"requester_uid": data.RequesterUID,
		"model_id":      data.ModelID,
		"model_uid":     data.ModelUID,
	}

	fields := map[string]any{
		"model_trigger_uid":     data.TriggerUID,
		"trigger_time":          data.TriggerTime,
		"compute_time_duration": data.ComputeTimeDuration,
	}

	return influxdb2.NewPoint(modelMeasurement, tags, fields, time.Now())
}
