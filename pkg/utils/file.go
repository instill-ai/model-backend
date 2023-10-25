package utils

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"gorm.io/datatypes"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
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
	var modelName string
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

		// TODO: version mapping?
		fPath = filepath.Join(dstDir, owner, uploadedModel.ID, "latest", f.Name)

		if strings.Contains(f.Name, "README.md") {
			readmeFilePath = fPath
		} else {
			modelName = filepath.Join(owner, uploadedModel.ID, "latest", f.Name)
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
	}

	uploadedModel.InferenceModels = []datamodel.InferenceModel{
		{
			Name:     modelName,
			Platform: "onnx",
			Version:  1,
		},
	}

	return readmeFilePath, fPath, nil
}

// modelDir and dstDir are absolute path
func UpdateModelPath(modelDir string, dstDir string, owner string, model *datamodel.Model) (string, string, error) {

	return "", "", nil
}

func SaveFile(stream modelPB.ModelPublicService_CreateUserModelBinaryFileUploadServer) (outFile string, parent string, modelInfo *datamodel.Model, modelDefinitionID string, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateUserModelBinaryFileUploadRequest

	var tmpFile string

	var uploadedModel datamodel.Model
	for {
		fileData, err = stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { //first chunk contains file name
			if fileData.Model == nil {
				return "", "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
			}

			if fileData.Parent == "" {
				return "", "", &datamodel.Model{}, "", fmt.Errorf("failed namespace parsing")
			}

			parent = fileData.Parent

			rdid, _ := uuid.NewV4()
			tmpFile = path.Join("/tmp", rdid.String()+".zip")
			fp, _ = os.Create(tmpFile)
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
				return "", "", &datamodel.Model{}, "", err
			}
			uploadedModel = datamodel.Model{
				ID:         fileData.Model.Id,
				Visibility: datamodel.ModelVisibility(visibility),
				Description: sql.NullString{
					String: description,
					Valid:  true,
				},
				State:         datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
				Configuration: datatypes.JSON{},
			}
			if err != nil {
				return "", "", &datamodel.Model{}, "", err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = WriteToFp(fp, fileData.Content)
		if err != nil {
			return "", "", &datamodel.Model{}, "", err
		}
	}
	return tmpFile, parent, &uploadedModel, modelDefinitionID, nil
}

// GetJSON fetches the contents of the given URL
// and decodes it as JSON into the given result,
// which should be a pointer to the expected data.
func GetJSON(url string, result interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http.Get %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Get status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	return nil
}
