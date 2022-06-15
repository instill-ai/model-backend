package util

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/gernest/front"
	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/mitchellh/mapstructure"
	"gorm.io/datatypes"
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

func findDVCPath(dir string) string {
	dvcPath := ""
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.Name() == ".dvc" {
			dvcPath = path
		}
		return nil
	})
	return dvcPath
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
	err := cmd.Run()
	if err != nil {
		return err
	}
	dvcPath := findDVCPath(dir)
	if dvcPath != "" {
		cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s; dvc pull", dvcPath))
		err = cmd.Run()
		return err
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

func DoSupportBatch(configFilePath string) (bool, error) {
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	file, err := os.Open(configFilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	r, err := regexp.Compile(`max_batch_size:\s0`) // this can also be a regex
	if err != nil {
		return false, err
	}

	for scanner.Scan() {
		if r.MatchString(scanner.Text()) {
			return false, nil
		}
	}

	return true, nil
}

func writeCredential(credential datatypes.JSON) (string, error) {
	var gcsCredential datamodel.GCSCredential
	err := json.Unmarshal([]byte(credential), &gcsCredential)
	if err != nil {
		return "", err
	}
	file, _ := json.MarshalIndent(gcsCredential, "", " ")
	uid, _ := uuid.NewV4()
	credentialFile := fmt.Sprintf("/tmp/%v", uid.String())
	err = ioutil.WriteFile(credentialFile, file, 0644)
	if err != nil {
		return "", err
	}
	return credentialFile, nil
}

func ArtiVCGetTags(dir string, config datamodel.ArtiVCModelConfiguration) ([]string, error) {
	url := config.Url
	var cmd *exec.Cmd
	if strings.HasPrefix(url, "gs://") {
		credentialFile, err := writeCredential(config.Credential)
		if err != nil {
			return []string{}, err
		}
		cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s avc clone %s %s", credentialFile, url, dir))
		err = cmd.Run()
		if err != nil {
			return []string{}, err
		}

		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s ; GOOGLE_APPLICATION_CREDENTIALS=%s avc tag", dir, credentialFile)).Output()
		if err != nil {
			return []string{}, err
		}

		elems := strings.Split(string(out), "\n")
		tags := []string{}
		for _, tag := range elems {
			if strings.Trim(tag, " ") != "" {
				tags = append(tags, tag)
			}
		}
		_ = os.Remove(credentialFile)
		return tags, err
	} else {
		return []string{}, fmt.Errorf("not support url %v", url)
	}
}

func ArtiVCClone(dir string, modelConfig datamodel.ArtiVCModelConfiguration, instanceConfig datamodel.ArtiVCModelInstanceConfiguration) error {
	url := modelConfig.Url
	var cmd *exec.Cmd
	if strings.HasPrefix(url, "gs://") {
		credentialFile, err := writeCredential(modelConfig.Credential)
		if err != nil {
			return err
		}
		cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s avc get -o %s %s@%s", credentialFile, dir, url, instanceConfig.Tag))
		err = cmd.Run()
		if err != nil {
			return err
		}
		_ = os.Remove(credentialFile)
		return err
	} else {
		return fmt.Errorf("not support url %v", url)
	}
}
