package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gernest/front"
	"github.com/iancoleman/strcase"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/mitchellh/mapstructure"
)

type ModelMeta struct {
	Tags []string
	Task string
}

func GetModelMetaFromReadme(readmeFilePath string) (*ModelMeta, error) {
	if _, err := os.Stat(readmeFilePath); err != nil {
		return &ModelMeta{}, err
	}
	file, err := os.Open(readmeFilePath)
	if err != nil {
		return &ModelMeta{}, err
	}
	fm := front.NewMatter()
	fm.Handle("---", front.YAMLHandler)
	meta, _, err := fm.Parse(file)
	if err != nil {
		return &ModelMeta{}, err
	}
	var modelMeta ModelMeta
	err = mapstructure.Decode(meta, &modelMeta)

	return &modelMeta, err
}

func GitHubClone(dir string, instanceConfig datamodel.GitHubModelInstanceConfiguration) error {
	urlRepo := instanceConfig.Repository
	if !strings.HasPrefix(urlRepo, "https://github.com") {
		urlRepo = "https://github.com/" + urlRepo
	}
	if !strings.HasSuffix(urlRepo, ".git") {
		urlRepo = urlRepo + ".git"
	}

	cmd := exec.Command("git", "clone", "-b", instanceConfig.Tag, urlRepo, dir)
	return cmd.Run()
}

type Tag struct {
	Name string `json:"name"`
}
type GitHubInfo struct {
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Tags        []Tag
}

func GetGitHubRepoInfo(repo string) (GitHubInfo, error) {
	if repo == "" {
		return GitHubInfo{}, fmt.Errorf("invalid repo URL")
	}

	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%v", repo))
	if err != nil {
		return GitHubInfo{}, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GitHubInfo{}, err
	}
	githubRepoInfo := GitHubInfo{}
	err = json.Unmarshal(body, &githubRepoInfo)
	if err != nil {
		return GitHubInfo{}, err
	}

	resp, err = http.Get(fmt.Sprintf("https://api.github.com/repos/%v/tags", repo))
	if err != nil {
		return GitHubInfo{}, err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return GitHubInfo{}, err
	}

	tags := []Tag{}
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return GitHubInfo{}, err
	}
	githubRepoInfo.Tags = tags

	return githubRepoInfo, nil
}

func RemoveModelRepository(modelRepositoryRoot string, namespace string, modelName string, instanceName string) {
	path := fmt.Sprintf("%v/%v#%v#*#%v", modelRepositoryRoot, namespace, modelName, instanceName)
	files, err := filepath.Glob(path)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			panic(err)
		}
	}
	readmeFilePath := fmt.Sprintf("%v/%v#%v#README.md#%v", modelRepositoryRoot, namespace, modelName, instanceName)
	_ = os.Remove(readmeFilePath)
}

// ConvertAllJSONKeySnakeCase traverses a JSON object to replace all keys to snake_case.
func ConvertAllJSONKeySnakeCase(i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			sc := strcase.ToSnake(k)
			if sc != k {
				v[sc] = v[k]
				delete(v, k)
			}
			ConvertAllJSONKeySnakeCase(vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertAllJSONKeySnakeCase(vv)
		}
	}
}

// ConvertAllJSONEnumValueToProtoStyle converts lowercase enum value to the Protobuf naming convention where the enum type is always prefixed and is UPPERCASE snake_case.
// For examples:
// - api in a Protobuf `Enum SourceType` type will be converted to SOURCE_TYPE_API
// - oauth2.0  in a Protobuf `Enum AuthFlowType` type will be converted to AUTH_FLOW_TYPE_OAUTH2_0
func ConvertAllJSONEnumValueToProtoStyle(enumRegistry map[string]map[string]int32, i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			if _, ok := enumRegistry[k]; ok {
				for enumKey := range enumRegistry[k] {
					if reflect.TypeOf(vv).Kind() == reflect.Slice { // repeated enum type
						for kk, vvv := range vv.([]interface{}) {
							if strings.ReplaceAll(vvv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
								vv.([]interface{})[kk] = enumKey
							}
						}
					} else {
						if strings.ReplaceAll(vv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
							v[k] = enumKey
						}
					}
				}
			}
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	}
}
