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
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gernest/front"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
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

func GetModelMetaFromReadme(readmeFilePath string) (*ModelMeta, error) {
	if err := ValidateFilePath(readmeFilePath); err != nil {
		return &ModelMeta{}, err
	}
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

func findDVCPaths(dir string) []string {
	dvcPaths := []string{}
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".dvc") {
			dvcPaths = append(dvcPaths, path)
		}
		return nil
	})
	return dvcPaths
}

func FindModelPythonDir(dir string) (modelPythonDir string) {
	modelPythonDir = ""
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() && f.Name() == "model.py" {
			modelPythonDir = filepath.Dir(path)
		}
		return nil
	})
	return modelPythonDir
}

// TODO: clean up this function.
func findModelFiles(dir string) []string {
	var modelPaths = []string{}
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(f.Name(), ".onnx") || strings.HasSuffix(f.Name(), ".pt") || strings.HasSuffix(f.Name(), ".bias") ||
			strings.HasSuffix(f.Name(), ".weight") || strings.HasSuffix(f.Name(), ".ini") || strings.HasSuffix(f.Name(), ".bin") ||
			strings.HasPrefix(f.Name(), "onnx__") || strings.HasSuffix(f.Name(), ".model") || strings.HasSuffix(f.Name(), ".json") ||
			strings.HasSuffix(f.Name(), ".xml") || strings.HasSuffix(f.Name(), ".msgpack") || strings.HasSuffix(f.Name(), ".onnx_data") ||
			strings.HasSuffix(f.Name(), ".txt") || strings.HasSuffix(f.Name(), ".safetensors") {
			modelPaths = append(modelPaths, path)
		}
		return nil
	})
	return modelPaths
}

type CacheModel struct {
	ModelRepo string `json:"model_repo"`
	State     string `json:"state"`
}

// GitHubClone clones a repository from GitHub.
func GitHubClone(dir string, instanceConfig datamodel.GitHubModelConfiguration, isWithLargeFile bool, redisClient *redis.Client) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlRepo := instanceConfig.Repository
	redisRepoKey := fmt.Sprintf("model_cache:%s:%s", instanceConfig.Repository, instanceConfig.Tag)
	// Check in the cache first.
	if config.Config.Cache.Model.Enabled {
		_ = os.MkdirAll(config.Config.Cache.Model.CacheDir, os.ModePerm)
		if state, err := redisClient.Get(ctx, redisRepoKey).Result(); err != nil && !errors.Is(err, redis.Nil) {
			return err
		} else if err == nil {
			if state == "done" {
				return nil
			} else if state == "without_large_file" && !isWithLargeFile {
				return nil
			}
		}
	}
	if !isWithLargeFile || isWithLargeFile && !config.Config.Cache.Model.Enabled {
		if !strings.HasPrefix(urlRepo, "https://github.com") {
			if config.Config.Github.PatEnabled {
				urlRepo = fmt.Sprintf("https://%s@github.com/%s", config.Config.Github.Pat, urlRepo)
				fmt.Println(urlRepo)
			} else {
				urlRepo = fmt.Sprintf("https://github.com/%s", urlRepo)
			}
		}
		if !strings.HasSuffix(urlRepo, ".git") {
			urlRepo += ".git"
		}

		extraFlag := ""
		if !isWithLargeFile {
			extraFlag = "GIT_LFS_SKIP_SMUDGE=1"
		}
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s git clone -b %s %s %s", extraFlag, instanceConfig.Tag, urlRepo, dir))
		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	var err error
	if isWithLargeFile {
		dvcPaths := findDVCPaths(dir)
		for _, dvcPath := range dvcPaths {
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s; dvc pull %s", dir, dvcPath))
			err = cmd.Run()
			if err != nil {
				return err
			}
		}
		if config.Config.Cache.Model.Enabled {
			redisClient.Set(ctx, redisRepoKey, "done", time.Duration(0))
		}
	} else if config.Config.Cache.Model.Enabled {
		redisClient.Set(ctx, redisRepoKey, "without_large_file", time.Duration(0))
	}

	return nil
}

// CopyModelFileToModelRepository copies model files to model repository.
func CopyModelFileToModelRepository(modelRepository string, dir string, model *datamodel.Model) error {
	modelDir := filepath.Join(modelRepository, model.Owner, model.ID)
	srcPath := fmt.Sprintf("%s/*", dir)

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -ru %s %s", srcPath, modelDir))
	if err := cmd.Run(); err != nil {
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
	if config.Config.Github.PatEnabled {
		repoRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Config.Github.Pat))
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
	if config.Config.Github.PatEnabled {
		tagRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Config.Github.Pat))
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

func HasModelInModelRepository(modelRepositoryRoot string, owner string, modelID string) bool {
	modelPath := filepath.Join(config.Config.RayServer.ModelStore, owner, modelID)

	if matches, _ := filepath.Glob(modelPath); matches == nil {
		return false
	}

	return true
}

func RemoveModelRepository(modelRepositoryRoot string, owner string, modelID string) {
	modelPath := filepath.Join(modelRepositoryRoot, owner, modelID)

	if err := ValidateFilePath(modelPath); err != nil {
		panic(err)
	}

	_ = os.RemoveAll(modelPath)
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

func ArtiVCGetTags(dir string, modelConfig datamodel.ArtiVCModelConfiguration) ([]string, error) {
	url := modelConfig.URL
	var cmd *exec.Cmd
	if strings.HasPrefix(url, "gs://") {
		credentialFile, err := writeCredential(modelConfig.Credential)
		if err != nil {
			return []string{}, err
		}
		cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s avc clone %s %s", credentialFile, url, dir))
		err = cmd.Run()
		if err != nil {
			_ = os.Remove(credentialFile)
			return []string{}, err
		}

		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s ; GOOGLE_APPLICATION_CREDENTIALS=%s avc tag", dir, credentialFile)).Output()
		if err != nil {
			_ = os.Remove(credentialFile)
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

func ArtiVCClone(dir string, modelConfig datamodel.ArtiVCModelConfiguration, withLargeFiles bool) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	url := modelConfig.URL
	var cmd *exec.Cmd
	if strings.HasPrefix(url, "gs://") {
		credentialFile, err := writeCredential(modelConfig.Credential)
		if err != nil {
			return err
		}
		if !withLargeFiles {
			// make artivc ignore large file
			cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("cp assets/artivc/avcignore %s/.avcignore", dir))
			err = cmd.Run()
			if err != nil {
				return err
			}
		}

		// download other source file such as .py, config.pbtxt
		cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s avc get -o %s %s@%s", credentialFile, dir, url, modelConfig.Tag))
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

func HuggingFaceClone(dir string, modelConfig datamodel.HuggingFaceModelConfiguration) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("GIT_LFS_SKIP_SMUDGE=1 git clone https://huggingface.co/%s %s", modelConfig.RepoID, dir))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func HuggingFaceExport(dir string, modelConfig datamodel.HuggingFaceModelConfiguration, modelID string) error {
	// export model to folder structure similar with inference to support copy the model into model repository later
	if err := os.MkdirAll(fmt.Sprintf("%s/%s-infer/1", dir, modelID), os.ModePerm); err != nil {
		return err
	}
	// atol 0.001 mean that accept difference with 0.1%
	cmd := exec.Command("/ray-conda/bin/python",
		"-m", "transformers.onnx",
		"--feature=image-classification",
		"--atol", "0.001",
		fmt.Sprintf("--model=%s", modelConfig.RepoID),
		fmt.Sprintf("%s/%s-infer/1", dir, modelID))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func UpdateConfigModelName(filePath string, oldModelName string, newModelName string) error {
	regStr := fmt.Sprintf("name:\\s+\"%v\"", oldModelName)
	nameRegx := regexp.MustCompile(regStr)
	if err := ValidateFilePath(filePath); err != nil {
		return err
	}
	fileData, _ := os.ReadFile(filePath)
	fileString := string(fileData)
	fileString = nameRegx.ReplaceAllString(fileString, fmt.Sprintf("name: \"%v\"", newModelName))
	fileData = []byte(fileString)
	return os.WriteFile(filePath, fileData, 0o600)
}

func UpdateModelName(filePath string, oldModelName string, newModelName string) error {
	nameRegx := regexp.MustCompile(oldModelName)
	if err := ValidateFilePath(filePath); err != nil {
		return err
	}
	fileData, _ := os.ReadFile(filePath)
	fileString := string(fileData)
	fileString = nameRegx.ReplaceAllString(fileString, newModelName)
	fileData = []byte(fileString)
	return os.WriteFile(filePath, fileData, 0o600)
}

func GenerateHuggingFaceModel(confDir string, dest string, modelID string) error {
	if err := os.Mkdir(dest, os.ModePerm); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -rf assets/huggingface-vit-template/* %s", dest))
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("mv %s/huggingface %s/%s", dest, dest, modelID))
	if err := cmd.Run(); err != nil {
		return err
	}

	configEnsemblePath := fmt.Sprintf("%s/%s/config.pbtxt", dest, modelID)
	if err := UpdateConfigModelName(configEnsemblePath, "huggingface", modelID); err != nil {
		return err
	}
	if err := UpdateConfigModelName(configEnsemblePath, "huggingface-infer", fmt.Sprintf("%s-infer", modelID)); err != nil {
		return err
	}

	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("mv %s/huggingface-infer %s/%s-infer", dest, dest, modelID))
	if err := cmd.Run(); err != nil {
		return err
	}
	configModelInferPath := fmt.Sprintf("%s/%s-infer/config.pbtxt", dest, modelID)
	if err := UpdateConfigModelName(configModelInferPath, "huggingface-infer", fmt.Sprintf("%s-infer", modelID)); err != nil {
		return err
	}

	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("cp %s/*.json %s/pre/1", confDir, dest))
	if err := cmd.Run(); err != nil {
		return err
	}

	if _, err := os.Stat(fmt.Sprintf("%s/README.md", confDir)); err != nil {
		return errors.New("there is no README file")
	}

	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("cp %s/README.md %s/", confDir, dest))
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func HasModelWeightFile(modelRepository string, model *datamodel.Model) bool {
	modelDir := filepath.Join(modelRepository, model.Owner, model.ID)
	modelFiles := findModelFiles(modelDir)
	if len(modelFiles) > 0 {
		for _, modelFile := range modelFiles {
			fi, _ := os.Stat(modelFile)
			if !strings.HasSuffix(fi.Name(), ".json") &&
				!strings.HasSuffix(fi.Name(), ".txt") &&
				!strings.HasSuffix(fi.Name(), ".xml") &&
				fi.Size() < 200 { // 200b
				return false
			}
		}
		return true
	}
	return false
}

func HasDVCWeightFile(modelRepository string, model *datamodel.Model) bool {
	modelDir := filepath.Join(modelRepository, model.Owner, model.ID)
	dvcFiles := findDVCPaths(modelDir)
	if len(dvcFiles) > 0 {
		for _, dvcFile := range dvcFiles {
			if _, err := os.Stat(strings.TrimSuffix(dvcFile, ".dvc")); err != nil {
				return false
			}
		}
		return true
	}
	return false
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
