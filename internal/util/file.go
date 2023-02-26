package util

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"gorm.io/datatypes"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

type FileMeta struct {
	path  string
	fInfo os.FileInfo
}

func isEnsembleConfig(configPath string) bool {
	fileData, _ := os.ReadFile(configPath)
	fileString := string(fileData)
	return strings.Contains(fileString, "platform: \"ensemble\"")
}

// TODO: should have other approach
func couldBeEnsembleConfig(configPath string) bool {
	fileData, _ := os.ReadFile(configPath)
	fileString := string(fileData)
	return strings.Contains(fileString, "instance_group") && strings.Contains(fileString, "backend: \"python\"")
}

// writeToFp takes in a file pointer and byte array and writes the byte array into the file
// returns error if pointer is nil or error in writing to file
func WriteToFp(fp *os.File, data []byte) error {
	w := 0
	n := len(data)
	for {

		nw, err := fp.Write(data[w:])
		if err != nil {
			return err
		}
		w += nw
		if nw >= n {
			return nil
		}
	}
}

// TODO: need to clean up this function
func Unzip(fPath string, dstDir string, owner string, uploadedModel *datamodel.Model) (string, string, error) {
	archive, err := zip.OpenReader(fPath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return "", "", err
	}
	defer archive.Close()
	var readmeFilePath string
	var createdTModels []datamodel.TritonModel
	var currentNewModelName string
	var currentOldModelName string
	var ensembleFilePath string
	var newModelNameMap = make(map[string]string)
	var configFiles []string
	for _, f := range archive.File {
		if strings.Contains(f.Name, "__MACOSX") || strings.Contains(f.Name, "__pycache__") { // ignore temp directory in macos
			continue
		}
		fPath := filepath.Join(dstDir, f.Name)
		fmt.Println("unzipping file ", fPath)

		if !strings.HasPrefix(fPath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return "", "", fmt.Errorf("invalid file path")
		}
		if f.FileInfo().IsDir() {
			dirName := f.Name
			if string(dirName[len(dirName)-1]) == "/" {
				dirName = dirName[:len(dirName)-1]
			}
			if !strings.Contains(dirName, "/") { // top directory model
				currentOldModelName = dirName
				dirName = fmt.Sprintf("%v#%v#%v#%v", owner, uploadedModel.ID, dirName, uploadedModel.Instances[0].ID)
				currentNewModelName = dirName
				newModelNameMap[currentOldModelName] = currentNewModelName
			} else { // version folder
				dirName = strings.Replace(dirName, currentOldModelName, currentNewModelName, 1)
				patternVersionFolder := fmt.Sprintf("^%v/[0-9]+$", currentNewModelName)
				match, _ := regexp.MatchString(patternVersionFolder, dirName)
				if match {
					elems := strings.Split(dirName, "/")
					sVersion := elems[len(elems)-1]
					iVersion, err := strconv.ParseInt(sVersion, 10, 32)
					if err == nil {
						createdTModels = append(createdTModels, datamodel.TritonModel{
							Name:    currentNewModelName, // Triton model name
							State:   datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
							Version: int(iVersion),
						})
					}
				}
			}
			fPath := filepath.Join(dstDir, dirName)
			if err := ValidateFilePath(fPath); err != nil {
				return "", "", err
			}
			err = os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return "", "", err
			}
			continue
		}

		// Update triton folder into format {model_name}#{task_name}#{task_version}
		subStrs := strings.Split(f.Name, "/")
		if len(subStrs) < 1 {
			continue
		}
		// Triton modelname is folder name
		oldModelName := subStrs[0]
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", owner, uploadedModel.ID, subStrs[0], uploadedModel.Instances[0].ID)
		newModelName := subStrs[0]
		fPath = filepath.Join(dstDir, strings.Join(subStrs, "/"))
		if strings.Contains(f.Name, "README.md") {
			readmeFilePath = fPath
		}
		if err := ValidateFilePath(fPath); err != nil {
			return "", "", err
		}
		// ensure the parent folder existed
		if _, err := os.Stat(filepath.Dir(fPath)); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return "", "", err
			}
		}

		dstFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", "", err
		}
		fileInArchive, err := f.Open()
		if err != nil {
			return "", "", err
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return "", "", err
		}

		dstFile.Close()
		fileInArchive.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(fPath)
		if fileExtension == ".pbtxt" {
			configFiles = append(configFiles, fPath)
			if isEnsembleConfig(fPath) {
				ensembleFilePath = fPath
			}
			err = UpdateConfigModelName(fPath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
	}
	if ensembleFilePath == "" {
		for _, filePath := range configFiles {
			if couldBeEnsembleConfig(filePath) {
				ensembleFilePath = filePath
				break
			}
		}

		for oldModelName, newModelName := range newModelNameMap {
			err = UpdateModelName(filepath.Dir(ensembleFilePath)+"/1/model.py", oldModelName, newModelName) // TODO: replace in all files.
			if err != nil {
				return "", "", err
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = UpdateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	uploadedModel.Instances[0].TritonModels = createdTModels
	return readmeFilePath, ensembleFilePath, nil
}

// modelDir and dstDir are absolute path
func UpdateModelPath(modelDir string, dstDir string, owner string, modelID string, modelInstance *datamodel.ModelInstance) (string, string, error) {
	var createdTModels []datamodel.TritonModel
	var ensembleFilePath string
	var newModelNameMap = make(map[string]string)
	var readmeFilePath string
	files := []FileMeta{}
	var configFiles []string
	var fileRe = regexp.MustCompile(`.git|.dvc|.dvcignore`)
	err := filepath.Walk(modelDir, func(path string, f os.FileInfo, err error) error {
		if !fileRe.MatchString(path) {
			files = append(files, FileMeta{
				path:  path,
				fInfo: f,
			})
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	modelRootDir := strings.Join([]string{dstDir, owner}, "/")
	err = os.MkdirAll(modelRootDir, os.ModePerm)
	if err != nil {
		return "", "", err
	}
	for _, f := range files {
		if f.path == modelDir {
			continue
		}
		// Update triton folder into format {model_name}#{task_name}#{task_version}
		subStrs := strings.Split(strings.Replace(f.path, modelDir+"/", "", 1), "/")
		if len(subStrs) < 1 {
			continue
		}
		// Triton modelname is folder name
		oldModelName := subStrs[0]
		subStrs[0] = fmt.Sprintf("%v#%v#%v#%v", owner, modelID, oldModelName, modelInstance.ID)
		var filePath = filepath.Join(dstDir, strings.Join(subStrs, "/"))

		if f.fInfo.IsDir() { // create new folder
			err = os.MkdirAll(filePath, os.ModePerm)

			if err != nil {
				return "", "", err
			}
			newModelNameMap[oldModelName] = subStrs[0]
			if v, err := strconv.Atoi(subStrs[len(subStrs)-1]); err == nil {
				createdTModels = append(createdTModels, datamodel.TritonModel{
					Name:    subStrs[0], // Triton model name
					State:   datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
					Version: int(v),
				})
			}
			continue
		}
		if strings.Contains(filePath, "README") {
			readmeFilePath = filePath
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.fInfo.Mode())
		if err != nil {
			return "", "", err
		}
		srcFile, err := os.Open(f.path)
		if err != nil {
			return "", "", err
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return "", "", err
		}
		dstFile.Close()
		srcFile.Close()
		// Update ModelName in config.pbtxt
		fileExtension := filepath.Ext(filePath)
		if fileExtension == ".pbtxt" {
			configFiles = append(configFiles, filePath)
			if isEnsembleConfig(filePath) {
				ensembleFilePath = filePath
			}
			err = UpdateConfigModelName(filePath, oldModelName, subStrs[0])
			if err != nil {
				return "", "", err
			}
		}
	}
	if ensembleFilePath == "" {
		for _, filePath := range configFiles {
			if couldBeEnsembleConfig(filePath) {
				ensembleFilePath = filePath
				break
			}
		}

		for oldModelName, newModelName := range newModelNameMap {
			err = UpdateModelName(filepath.Dir(ensembleFilePath)+"/1/model.py", oldModelName, newModelName) // TODO: replace in all files.
			if err != nil {
				return "", "", err
			}
		}
	}
	// Update ModelName in ensemble model config file
	if ensembleFilePath != "" {
		for oldModelName, newModelName := range newModelNameMap {
			err = UpdateConfigModelName(ensembleFilePath, oldModelName, newModelName)
			if err != nil {
				return "", "", err
			}
		}
		for i := 0; i < len(createdTModels); i++ {
			if strings.Contains(ensembleFilePath, createdTModels[i].Name) {
				createdTModels[i].Platform = "ensemble"
				break
			}
		}
	}
	modelInstance.TritonModels = createdTModels
	return readmeFilePath, ensembleFilePath, nil
}

func SaveFile(stream modelPB.ModelService_CreateModelBinaryFileUploadServer) (outFile string, modelInfo *datamodel.Model, modelDefinitionID string, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateModelBinaryFileUploadRequest

	var tmpFile string

	var uploadedModel datamodel.Model
	for {
		fileData, err = stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { //first chunk contains file name
			if fileData.Model == nil {
				return "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
			}

			rdid, _ := uuid.NewV4()
			tmpFile = path.Join("/tmp", rdid.String()+".zip")
			fp, err = os.Create(tmpFile)
			visibility := modelPB.Model_VISIBILITY_PRIVATE
			if fileData.Model.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
				visibility = modelPB.Model_VISIBILITY_PUBLIC
			}
			var description = ""
			if fileData.Model.Description != nil {
				description = *fileData.Model.Description
			}
			modelDefName := fileData.Model.ModelDefinition
			modelDefinitionID, err = resource.GetDefinitionID(modelDefName)
			if err != nil {
				return "", &datamodel.Model{}, "", err
			}
			uploadedModel = datamodel.Model{
				ID:         fileData.Model.Id,
				Visibility: datamodel.ModelVisibility(visibility),
				Description: sql.NullString{
					String: description,
					Valid:  true,
				},
				Configuration: datatypes.JSON{},
				Instances: []datamodel.ModelInstance{{
					State: datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE),
					ID:    "latest",
				}},
			}
			if err != nil {
				return "", &datamodel.Model{}, "", err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = WriteToFp(fp, fileData.Content)
		if err != nil {
			return "", &datamodel.Model{}, "", err
		}
	}
	return tmpFile, &uploadedModel, modelDefinitionID, nil
}
