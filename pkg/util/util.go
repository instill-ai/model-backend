package util

import (
	"bufio"
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
	"sort"
	"strconv"
	"strings"

	"github.com/gernest/front"
	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
	"gorm.io/datatypes"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

type ModelMeta struct {
	Tags []string
	Task string
}

// validate to prevent security issue as https://codeql.github.com/codeql-query-help/go/go-path-injection/
func ValidateFilePath(filePath string) error {
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("the deleted file should not contain special characters")
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

// TODO: clean up this function.
func findModelFiles(dir string) []string {
	var modelPaths []string = []string{}
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(f.Name(), ".onnx") || strings.HasSuffix(f.Name(), ".pt") || strings.HasSuffix(f.Name(), ".bias") ||
			strings.HasSuffix(f.Name(), ".weight") || strings.HasSuffix(f.Name(), ".ini") || strings.HasSuffix(f.Name(), ".bin") ||
			strings.HasPrefix(f.Name(), "onnx__") {
			modelPaths = append(modelPaths, path)
		}
		return nil
	})
	return modelPaths
}

func AddMissingTritonModelFolder(dir string) {
	logger, _ := logger.GetZapLogger()
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.Name() == "config.pbtxt" {
			if _, err := os.Stat(fmt.Sprintf("%s/1", filepath.Dir(path))); err != nil {
				if err := os.MkdirAll(fmt.Sprintf("%s/1", filepath.Dir(path)), os.ModePerm); err != nil {
					logger.Error(err.Error())
				}
			}
		}
		return nil
	})
}

func getPreModelConfigPath(modelRepository string, tritonModels []datamodel.TritonModel) string {
	modelPath := ""
	for _, triton := range tritonModels {
		if strings.Contains(triton.Name, "#pre#") {
			return fmt.Sprintf("%s/%s", modelRepository, triton.Name)
		}
	}
	return modelPath
}
func getInferModelConfigPath(modelRepository string, tritonModels []datamodel.TritonModel) string {
	modelPath := ""
	for _, triton := range tritonModels {
		if strings.Contains(triton.Name, "-infer#") {
			return fmt.Sprintf("%s/%s", modelRepository, triton.Name)
		}
	}
	return modelPath
}

type CacheModel struct {
	ModelRepo string `json:"model_repo"`
	State     string `json:"state"`
}

// GitHubClone clones a repository from GitHub.
func GitHubClone(dir string, instanceConfig datamodel.GitHubModelConfiguration, isWithLargeFile bool) error {
	urlRepo := instanceConfig.Repository

	// Check in the cache first.
	var cacheModels []CacheModel
	if config.Config.Cache.Model {
		_ = os.MkdirAll(MODEL_CACHE_DIR, os.ModePerm)
		if _, err := os.Stat(MODEL_CACHE_DIR + "/" + MODEL_CACHE_FILE); !os.IsNotExist(err) {
			f, err := os.ReadFile(MODEL_CACHE_DIR + "/" + MODEL_CACHE_FILE)
			if err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(f), &cacheModels); err != nil {
				return err
			}
			for _, cacheModel := range cacheModels {
				if cacheModel.ModelRepo != (instanceConfig.Repository + instanceConfig.Tag) {
					continue
				}
				if cacheModel.State == "done" { // everything is cached.
					return nil
				} else if cacheModel.State == "without_large_file" && !isWithLargeFile { // the GitHub repo is being cached.
					return nil
				}
			}
		}
	}
	if !isWithLargeFile || isWithLargeFile && !config.Config.Cache.Model {
		if !strings.HasPrefix(urlRepo, "https://github.com") {
			urlRepo = "https://github.com/" + urlRepo
		}
		if !strings.HasSuffix(urlRepo, ".git") {
			urlRepo = urlRepo + ".git"
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

	var f *os.File
	var err error
	if config.Config.Cache.Model {
		f, err = os.Create(MODEL_CACHE_DIR + "/" + MODEL_CACHE_FILE)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	if isWithLargeFile {
		dvcPaths := findDVCPaths(dir)
		for _, dvcPath := range dvcPaths {
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s; dvc pull %s", dir, dvcPath))
			err = cmd.Run()
			if err != nil {
				return err
			}
		}
		if config.Config.Cache.Model {
			for i, cacheModel := range cacheModels {
				if cacheModel.ModelRepo == (instanceConfig.Repository + instanceConfig.Tag) {
					cacheModels[i].State = "done"
					break
				}
			}
			b, err := json.Marshal(cacheModels)
			if err != nil {
				return err
			}
			if _, err := f.Write(b); err != nil {
				return err
			}
		}
	} else {
		if config.Config.Cache.Model {
			cacheFile := CacheModel{
				ModelRepo: instanceConfig.Repository + instanceConfig.Tag,
				State:     "without_large_file",
			}
			cacheModels = append(cacheModels, cacheFile)
			b, err := json.Marshal(cacheModels)
			if err != nil {
				return err
			}
			if _, err := f.Write(b); err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyModelFileToModelRepository copies model files to model repository.
func CopyModelFileToModelRepository(modelRepository string, dir string, tritonModels []datamodel.TritonModel) error {
	modelPaths := findModelFiles(dir)
	for _, modelPath := range modelPaths {
		folderModelDir := filepath.Dir(modelPath)
		modelSubNames := strings.Split(folderModelDir, "/")
		if len(modelSubNames) < 2 {
			continue
		}
		for _, tritonModel := range tritonModels {
			tritonModelName := tritonModel.Name
			tritonSubNames := strings.Split(tritonModelName, "#")
			if len(tritonSubNames) < 4 {
				continue
			}

			if tritonSubNames[len(tritonSubNames)-2] == modelSubNames[len(modelSubNames)-2] {
				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp %s %s/%s/1", modelPath, modelRepository, tritonModelName))
				if err := cmd.Run(); err != nil {
					return err
				}
				// TODO: add general function to check if backend use fastertransformer, which has different model file structure
			} else if modelSubNames[len(modelSubNames)-3] == "fastertransformer" && tritonSubNames[len(tritonSubNames)-2] == modelSubNames[len(modelSubNames)-3] {
				targetPath := fmt.Sprintf("%s/%s/%s/%s/", modelRepository, tritonModelName, modelSubNames[len(modelSubNames)-2], modelSubNames[len(modelSubNames)-1])
				if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
					return err
				}
				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp %s %s/", modelPath, targetPath))
				if err := cmd.Run(); err != nil {
					return err
				}
			}
		}
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
	if repo == "" {
		return &GitHubInfo{}, fmt.Errorf("invalid repo URL")
	}

	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%v", repo))
	if err != nil {
		return &GitHubInfo{}, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &GitHubInfo{}, err
	}
	githubRepoInfo := GitHubInfo{}
	err = json.Unmarshal(body, &githubRepoInfo)
	if err != nil {
		return &GitHubInfo{}, err
	}
	resp, err = http.Get(fmt.Sprintf("https://api.github.com/repos/%v/tags", repo))
	if err != nil {
		return &GitHubInfo{}, err
	}
	body, err = io.ReadAll(resp.Body)
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

func RemoveModelRepository(modelRepositoryRoot string, namespace string, modelName string, instanceName string) {
	path := fmt.Sprintf("%v/%v#%v#*#%v", modelRepositoryRoot, namespace, modelName, instanceName)
	if err := ValidateFilePath(path); err != nil {
		panic(err)
	}

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
	if err := ValidateFilePath(readmeFilePath); err != nil {
		panic(err)
	}

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
	case map[string][]map[string]interface{}:
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
	r, err := regexp.Compile(`max_batch_size:`)
	if err != nil {
		return -1, err
	}

	for scanner.Scan() {
		if r.MatchString(scanner.Text()) {
			maxBatchSize := scanner.Text()
			maxBatchSize = strings.TrimPrefix(maxBatchSize, "max_batch_size:")
			maxBatchSize = strings.Trim(maxBatchSize, " ")
			intMaxBatchSize, err := strconv.Atoi(maxBatchSize)
			return intMaxBatchSize, err
		}
	}

	return -1, fmt.Errorf("not found")
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
	uid, _ := uuid.NewV4()
	credentialFile := fmt.Sprintf("/tmp/%v", uid.String())

	if credential == nil { // download default service account
		out, err := os.Create(credentialFile)
		if err != nil {
			return "", err
		}
		defer out.Close()
		resp, err := http.Get(DEFAULT_GCP_SERVICE_ACCOUNT_FILE)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

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
	url := modelConfig.Url
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
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("GIT_LFS_SKIP_SMUDGE=1 git clone https://huggingface.co/%s %s", modelConfig.RepoId, dir))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func HuggingFaceExport(dir string, modelConfig datamodel.HuggingFaceModelConfiguration, modelID string) error {
	// export model to folder structure similar with triton to support copy the model into model repository later
	if err := os.MkdirAll(fmt.Sprintf("%s/%s-infer/1", dir, modelID), os.ModePerm); err != nil {
		return err
	}
	// atol 0.001 mean that accept difference with 0.1%
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("python3 -m transformers.onnx --feature=image-classification --atol 0.001 --model=%s %s/%s-infer/1", modelConfig.RepoId, dir, modelID))
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
		return fmt.Errorf("there is no README file")
	}

	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("cp %s/README.md %s/", confDir, dest))
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func HasModelWeightFile(modelRepository string, tritonModels []datamodel.TritonModel) bool {
	for _, tritonModel := range tritonModels {
		modelDir := fmt.Sprintf("%s/%s", modelRepository, tritonModel.Name)
		modelFiles := findModelFiles(modelDir)
		if len(modelFiles) > 0 {
			for _, modelFile := range modelFiles {
				fi, _ := os.Stat(modelFile)
				if fi.Size() < 200 { // 200b
					return false
				}
			}
			return true
		}
	}
	return false
}

func updateModelConfigModel(configFilePath string, oldStr string, newStr string) error {
	if _, err := os.Stat(configFilePath); err != nil {
		return err
	}
	fileData, _ := os.ReadFile(configFilePath)
	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, oldStr, newStr)
	fileData = []byte(fileString)
	return os.WriteFile(configFilePath, fileData, 0o600)
}

func UpdateModelConfig(modelRepository string, tritonModels []datamodel.TritonModel) error {
	modelPathDir := getInferModelConfigPath(modelRepository, tritonModels)
	if modelPathDir == "" {
		return fmt.Errorf("there is no model")
	}
	modelFilePath := fmt.Sprintf("%s/1/model.onnx", modelPathDir)
	if _, err := os.Stat(modelFilePath); err != nil {
		return err
	}

	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("python3 assets/scripts/query_model_onnx.py -f %s", modelFilePath)).Output()
	if err != nil {
		return err
	}

	elems := strings.Split(string(out), ",")
	if len(elems) != 5 {
		return fmt.Errorf("wrong output format")
	}
	inputDim1, err := strconv.Atoi(elems[1])
	if err != nil {
		return err
	}
	inputDim2, err := strconv.Atoi(elems[2])
	if err != nil {
		return err
	}
	outputDim, err := strconv.Atoi(strings.TrimSuffix(elems[4], "\n"))
	if err != nil {
		return err
	}

	inferConfigFilePath := fmt.Sprintf("%s/config.pbtxt", modelPathDir)
	err = updateModelConfigModel(inferConfigFilePath,
		"dims: [ 3, 224, 224 ]",
		fmt.Sprintf("dims: [ 3, %v, %v ]", inputDim1, inputDim2))
	if err != nil {
		return err
	}

	err = updateModelConfigModel(inferConfigFilePath,
		"dims: [ -1 ]",
		fmt.Sprintf("dims: [ %v ]", outputDim))
	if err != nil {
		return err
	}

	preModelPathDir := getPreModelConfigPath(modelRepository, tritonModels)
	preConfigFilePath := fmt.Sprintf("%s/config.pbtxt", preModelPathDir)
	err = updateModelConfigModel(preConfigFilePath,
		"dims: [ 3, 224, 224 ]",
		fmt.Sprintf("dims: [ 3, %v, %v ]", inputDim1, inputDim2))
	if err != nil {
		return err
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/1/config.json", preModelPathDir))
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		return err
	}
	if id2label, ok := data["id2label"]; ok {
		mId2label := id2label.(map[string]interface{})

		keys := make([]int, 0, len(mId2label))
		for k := range mId2label {
			i, _ := strconv.Atoi(k)
			keys = append(keys, i)
		}
		sort.Ints(keys)

		f, err := os.Create(fmt.Sprintf("%s/label.txt", modelPathDir))
		if err != nil {
			return err
		}
		defer f.Close()

		for _, k := range keys {
			if _, err := f.WriteString(fmt.Sprintf("%s\n", mId2label[fmt.Sprintf("%v", k)])); err != nil {
				return err
			}
		}
		if err := updateModelConfigModel(inferConfigFilePath,
			fmt.Sprintf("dims: [ %v ]", outputDim),
			fmt.Sprintf("dims: [ %v ] \n label_filename: \"label.txt\"", outputDim)); err != nil {
			return err
		}
	}

	return nil
}

func GetSupportedBatchSize(task datamodel.ModelTask) int {
	allowedMaxBatchSize := 0
	switch task {
	case datamodel.ModelTask(modelPB.Model_TASK_UNSPECIFIED):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Unspecified
	case datamodel.ModelTask(modelPB.Model_TASK_CLASSIFICATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Classification
	case datamodel.ModelTask(modelPB.Model_TASK_DETECTION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Detection
	case datamodel.ModelTask(modelPB.Model_TASK_KEYPOINT):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Keypoint
	case datamodel.ModelTask(modelPB.Model_TASK_OCR):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.Ocr
	case datamodel.ModelTask(modelPB.Model_TASK_INSTANCE_SEGMENTATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.InstanceSegmentation
	case datamodel.ModelTask(modelPB.Model_TASK_SEMANTIC_SEGMENTATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.SemanticSegmentation
	case datamodel.ModelTask(modelPB.Model_TASK_TEXT_GENERATION):
		allowedMaxBatchSize = config.Config.MaxBatchSizeLimitation.TextGeneration
	}
	return allowedMaxBatchSize
}

func ConvertModelToResourcePermalink(modelUID string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/models", modelUID)

	return resourcePermalink
}
