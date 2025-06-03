package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/santhosh-tekuri/jsonschema/v5"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type ModelMeta struct {
	Tags []string
	Task string
}

type ModelSpec struct {
	ModelSpecSchema          *jsonschema.Schema `json:"model_schema,omitempty"`
	ModelConfigurationSchema *jsonschema.Schema `json:"configuration_schema,omitempty"`
}

// validate to prevent security issue as https://codeql.github.com/codeql-query-help/go/go-path-injection/
func ValidateFilePath(filePath string) error {
	if strings.Contains(filePath, "..") {
		return errors.New("the deleted file should not contain special characters")
	}
	return nil
}

type Tag struct {
	Name string `json:"name"`
}
type GitHubInfo struct {
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Tags        []Tag
}

func GetGitHubRepoInfo(repo string) (*GitHubInfo, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if repo == "" {
		return &GitHubInfo{}, errors.New("invalid repo URL")
	}

	repoRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%v", repo), http.NoBody)
	if err != nil {
		return &GitHubInfo{}, err
	}
	repoResp, err := http.DefaultClient.Do(repoRequest)
	if err != nil {
		return &GitHubInfo{}, err
	}
	defer repoResp.Body.Close()
	if repoResp.StatusCode != http.StatusOK {
		return &GitHubInfo{}, errors.New(repoResp.Status)
	}

	body, err := io.ReadAll(repoResp.Body)
	if err != nil {
		return &GitHubInfo{}, err
	}
	githubRepoInfo := GitHubInfo{}
	err = json.Unmarshal(body, &githubRepoInfo)
	if err != nil {
		return &GitHubInfo{}, err
	}

	tagRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%v/tags", repo), http.NoBody)
	if err != nil {
		return &GitHubInfo{}, err
	}
	tagResp, err := http.DefaultClient.Do(tagRequest)
	if err != nil {
		return &GitHubInfo{}, err
	}
	defer tagResp.Body.Close()
	if tagResp.StatusCode != http.StatusOK {
		return &GitHubInfo{}, errors.New(tagResp.Status)
	}

	body, err = io.ReadAll(tagResp.Body)
	if err != nil {
		return &GitHubInfo{}, err
	}

	tags := []Tag{}
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return &GitHubInfo{}, err
	}
	githubRepoInfo.Tags = tags

	return &githubRepoInfo, nil
}

// ConvertAllJSONKeySnakeCase traverses a JSON object to replace all keys to snake_case.
func ConvertAllJSONKeySnakeCase(i any) {

	switch v := i.(type) {
	case map[string]any:
		for k, vv := range v {
			sc := strcase.ToSnake(k)
			if sc != k {
				v[sc] = v[k]
				delete(v, k)
			}
			ConvertAllJSONKeySnakeCase(vv)
		}
	case []map[string]any:
		for _, vv := range v {
			ConvertAllJSONKeySnakeCase(vv)
		}
	case map[string][]map[string]any:
		for k, vv := range v {
			sc := strcase.ToSnake(k)
			if sc != k {
				v[sc] = v[k]
				delete(v, k)
			}
			ConvertAllJSONKeySnakeCase(vv)
		}
	}
}

// ConvertAllJSONEnumValueToProtoStyle converts lowercase enum value to the Protobuf naming convention where the enum type is always prefixed and is UPPERCASE snake_case.
// For examples:
// - api in a Protobuf `Enum SourceType` type will be converted to SOURCE_TYPE_API
// - oauth2.0  in a Protobuf `Enum AuthFlowType` type will be converted to AUTH_FLOW_TYPE_OAUTH2_0
func ConvertAllJSONEnumValueToProtoStyle(enumRegistry map[string]map[string]int32, i any) {
	switch v := i.(type) {
	case map[string]any:
		for k, vv := range v {
			if _, ok := enumRegistry[k]; ok {
				for enumKey := range enumRegistry[k] {
					if reflect.TypeOf(vv).Kind() == reflect.Slice { // repeated enum type
						for kk, vvv := range vv.([]any) {
							if strings.ReplaceAll(vvv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
								vv.([]any)[kk] = enumKey
							}
						}
					} else if strings.ReplaceAll(vv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
						v[k] = enumKey
					}
				}
			}
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	case []map[string]any:
		for _, vv := range v {
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	}
}

func GetMaxBatchSize(configFilePath string) (int, error) {
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		return -1, err
	}
	file, err := os.Open(configFilePath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	r := regexp.MustCompile(`max_batch_size:`)

	for scanner.Scan() {
		if !r.MatchString(scanner.Text()) {
			continue
		}
		maxBatchSize := scanner.Text()
		maxBatchSize = strings.TrimPrefix(maxBatchSize, "max_batch_size:")
		maxBatchSize = strings.Trim(maxBatchSize, " ")
		intMaxBatchSize, err := strconv.Atoi(maxBatchSize)
		return intMaxBatchSize, err
	}

	return -1, errors.New("not found")
}

// TODO: properly support batch inference
func DoSupportBatch() (bool, error) {
	return true, nil
}

func ConvertModelToResourcePermalink(modelUID string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/models", modelUID)

	return resourcePermalink
}

const (
	CreateEvent    string = "Create"
	UpdateEvent    string = "Update"
	DeleteEvent    string = "Delete"
	DeployEvent    string = "Deploy"
	UndeployEvent  string = "Undeploy"
	PublishEvent   string = "Publish"
	UnpublishEvent string = "Unpublish"
	TriggerEvent   string = "Trigger"
	TestEvent      string = "Test"

	modelMeasurement = "model.trigger.v1"
)

func IsAuditEvent(eventName string) bool {
	return strings.HasPrefix(eventName, CreateEvent) ||
		strings.HasPrefix(eventName, UpdateEvent) ||
		strings.HasPrefix(eventName, DeleteEvent) ||
		strings.HasPrefix(eventName, DeployEvent) ||
		strings.HasPrefix(eventName, UndeployEvent) ||
		strings.HasPrefix(eventName, PublishEvent) ||
		strings.HasPrefix(eventName, UnpublishEvent) ||
		strings.HasPrefix(eventName, TriggerEvent) ||
		strings.HasPrefix(eventName, TestEvent)
}

// TODO: billable event TBD
func IsBillableEvent(_ string) bool {
	return false
}

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
