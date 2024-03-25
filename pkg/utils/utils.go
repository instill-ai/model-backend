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

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"gorm.io/datatypes"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type ModelMeta struct {
	Tags []string
	Task string
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

func writeCredential(credential datatypes.JSON) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uid, _ := uuid.NewV4()
	credentialFile := fmt.Sprintf("/tmp/%v", uid.String())

	if credential == nil { // download default service account
		out, err := os.Create(credentialFile)
		if err != nil {
			return "", err
		}
		defer out.Close()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultGCPServiceAccountFile, http.NoBody)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("http.Do with  MethodGet %q: %w", DefaultGCPServiceAccountFile, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("http.Do with  MethodGet status: %s", resp.Status)
		}

		if _, err := io.Copy(out, resp.Body); err != nil {
			return "", err
		}
	} else {
		var gcsUserAccountCredential datamodel.GCSUserAccount
		if err := json.Unmarshal([]byte(credential), &gcsUserAccountCredential); err != nil {
			return "", err
		}

		file, err := json.MarshalIndent(gcsUserAccountCredential, "", " ")
		if err != nil {
			return "", err
		}
		// Validate GCSUserAccountJSONSchema JSON Schema
		if err := datamodel.ValidateJSONSchemaString(datamodel.GCSUserAccountJSONSchema, string(file)); err != nil {
			var gcsServiceAccountCredential datamodel.GCSServiceAccount
			if err := json.Unmarshal([]byte(credential), &gcsServiceAccountCredential); err != nil {
				return "", err
			}
			file, err := json.MarshalIndent(gcsServiceAccountCredential, "", " ")
			if err != nil {
				return "", err
			}
			// Validate GCSServiceAccountJSONSchema JSON Schema
			if err := datamodel.ValidateJSONSchemaString(datamodel.GCSServiceAccountJSONSchema, string(file)); err != nil {
				return "", err
			}
			if err := os.WriteFile(credentialFile, file, 0644); err != nil {
				return "", err
			}
		} else {
			if err := os.WriteFile(credentialFile, file, 0644); err != nil {
				return "", err
			}
		}
	}

	return credentialFile, nil
}


func GetSupportedBatchSize(task datamodel.ModelTask) int {
	allowedMaxBatchSize := 0
	switch task {
	case datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
	case datamodel.ModelTask(commonPB.Task_TASK_CLASSIFICATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
	case datamodel.ModelTask(commonPB.Task_TASK_DETECTION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
	case datamodel.ModelTask(commonPB.Task_TASK_KEYPOINT):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
	case datamodel.ModelTask(commonPB.Task_TASK_OCR):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
	case datamodel.ModelTask(commonPB.Task_TASK_INSTANCE_SEGMENTATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.InstanceSegmentation
	case datamodel.ModelTask(commonPB.Task_TASK_SEMANTIC_SEGMENTATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.SemanticSegmentation
	case datamodel.ModelTask(commonPB.Task_TASK_TEXT_GENERATION),
		datamodel.ModelTask(commonPB.Task_TASK_TEXT_GENERATION_CHAT),
		datamodel.ModelTask(commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.TextGeneration
	}
	return allowedMaxBatchSize
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
	OwnerUID           string
	OwnerType          mgmtPB.OwnerType
	UserUID            string
	UserType           mgmtPB.OwnerType
	ModelUID           string
	Status             mgmtPB.Status
	TriggerUID         string
	ModelDefinitionUID string
	TriggerTime        string
	ModelTask          commonPB.Task
}
