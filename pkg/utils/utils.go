package utils

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/santhosh-tekuri/jsonschema/v5"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
)

// ResourcePrefix represents the prefix for different resource types in AIP-compliant IDs
type ResourcePrefix string

const (
	PrefixModel           ResourcePrefix = "mod"
	PrefixModelDefinition ResourcePrefix = "mdf"
	PrefixModelVersion    ResourcePrefix = "ver"
	PrefixModelTag        ResourcePrefix = "mtg"
)

// base62Chars contains the characters used for base62 encoding (URL-safe without special chars)
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// encodeBase62 encodes a byte slice to a base62 string
func encodeBase62(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var result strings.Builder
	for _, b := range data {
		if b == 0 {
			result.WriteByte(base62Chars[0])
		} else {
			for b > 0 {
				result.WriteByte(base62Chars[b%62])
				b /= 62
			}
		}
	}
	return result.String()
}

// GeneratePrefixedResourceID creates an AIP-compliant prefixed resource ID from a UUID.
// The format is: {prefix}-{base62(sha256(uid)[:10])}
// This provides 80 bits of entropy in a URL-safe format.
func GeneratePrefixedResourceID(prefix ResourcePrefix, uid uuid.UUID) string {
	hash := sha256.Sum256([]byte(uid.String()))
	encoded := encodeBase62(hash[:10])
	return fmt.Sprintf("%s-%s", prefix, encoded)
}

// GenerateSlug converts a display name to a URL-safe slug.
// Example: "My ML Model" -> "my-ml-model"
func GenerateSlug(displayName string) string {
	slug := strings.ToLower(displayName)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	slug = re.ReplaceAllString(slug, "")
	multiDashRegex := regexp.MustCompile(`-+`)
	slug = multiDashRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

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
