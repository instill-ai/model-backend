package utils

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ProgressReader struct {
	r io.Reader

	filename   string
	n          float64
	lastPrintN float64
	lastPrint  time.Time
	logger     *zap.Logger
}

func NewProgressReader(r io.Reader, filename string) *ProgressReader {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := custom_logger.GetZapLogger(ctx)
	return &ProgressReader{
		r:        r,
		logger:   logger,
		filename: filename,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	bf := float64(n)
	bf /= (1 << 10)
	pr.n += bf

	if time.Since(pr.lastPrint) > time.Second ||
		(err != nil && pr.n != pr.lastPrintN) {

		pr.logger.Info(fmt.Sprintf("Copied %3.1fKiB for %s", pr.n, pr.filename))
		pr.lastPrintN = pr.n
		pr.lastPrint = time.Now()
	}
	return n, err
}

type FileMeta struct {
	path  string
	fInfo os.FileInfo
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
func Unzip(fPath string, dstDir string, owner string, uploadedModel *datamodel.Model) (string, error) {
	archive, err := zip.OpenReader(fPath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return "", err
	}
	defer archive.Close()
	var protoFilePath string
	var readmeFilePath string

	modelRootDir := filepath.Join(dstDir, owner, uploadedModel.ID)

	for _, f := range archive.File {
		if strings.Contains(f.Name, "__MACOSX") || strings.Contains(f.Name, "__pycache__") { // ignore temp directory in macos
			continue
		}
		fPath := filepath.Join(modelRootDir, f.Name)
		fmt.Println("unzipping file ", fPath)

		if !strings.HasPrefix(fPath, filepath.Clean(modelRootDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return "", fmt.Errorf("invalid file path")
		}

		if f.FileInfo().IsDir() {
			dirName := f.Name
			if string(dirName[len(dirName)-1]) == "/" {
				dirName = dirName[:len(dirName)-1]
			}
			fPath := filepath.Join(modelRootDir, dirName)
			if err := ValidateFilePath(fPath); err != nil {
				return "", err
			}
			err = os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return "", err
			}
			continue
		}

		if strings.Contains(f.Name, "README.md") {
			readmeFilePath = fPath
		}

		// ensure the parent folder existed
		if _, err := os.Stat(filepath.Dir(fPath)); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return "", err
			}
		}

		dstFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}
		fileInArchive, err := f.Open()
		if err != nil {
			return "", err
		}
		reader := NewProgressReader(fileInArchive, f.Name)
		if _, err := io.Copy(dstFile, reader); err != nil {
			return "", err
		}

		if err := dstFile.Close(); err != nil {
			return "", err
		}
		if err := fileInArchive.Close(); err != nil {
			return "", err
		}

		if filepath.Base(fPath) == "model.py" {
			protoFilePath = filepath.Dir(fPath)
		}
	}

	if err := copyRayProto(protoFilePath); err != nil {
		return "", err
	}

	_, err = filepath.Rel(config.Config.RayServer.ModelStore, readmeFilePath)
	if err != nil {
		return "", nil
	}

	return readmeFilePath, nil
}

// modelDir and dstDir are absolute path
func UpdateModelPath(modelDir string, dstDir string, owner string, model *datamodel.Model) (string, error) {
	var protoFilePath string
	var readmeFilePath string
	files := []FileMeta{}
	var fileRe = regexp.MustCompile(`/.git|/.dvc|/.dvcignore`)
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
		return "", err
	}
	modelRootDir := filepath.Join(dstDir, owner, model.ID)
	err = os.MkdirAll(modelRootDir, os.ModePerm)
	if err != nil {
		return "", err
	}
	var modelConfiguration datamodel.GitHubModelConfiguration
	_ = json.Unmarshal(model.Configuration, &modelConfiguration)
	for _, f := range files {
		if f.path == modelDir {
			continue
		}
		var filePath = filepath.Join(modelRootDir, f.path)

		if f.fInfo.IsDir() { // create new folder
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return "", err
			}
			continue
		}
		if strings.Contains(filePath, "README") {
			readmeFilePath = filePath
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.fInfo.Mode())
		if err != nil {
			return "", err
		}
		srcFile, err := os.Open(f.path)
		if err != nil {
			return "", err
		}
		reader := NewProgressReader(srcFile, f.fInfo.Name())
		if _, err := io.Copy(dstFile, reader); err != nil {
			return "", err
		}
		if err := dstFile.Close(); err != nil {
			return "", err
		}
		if err := srcFile.Close(); err != nil {
			return "", err
		}
		if filepath.Base(filePath) == "model.py" {
			protoFilePath = filepath.Dir(filePath)
		}
	}

	if err := copyRayProto(protoFilePath); err != nil {
		return "", err
	}

	return readmeFilePath, nil
}

type CreateNamespaceModelBinaryFileUploadRequestInterface interface {
	GetModel() *modelPB.Model
	GetParent() string
}

func SaveUserFile(stream modelPB.ModelPublicService_CreateUserModelBinaryFileUploadServer) (tmpFile string, parent string, uploadedModel *datamodel.Model, modelDefinitionID string, err error) {
	firstChunk := true
	var fileData *modelPB.CreateUserModelBinaryFileUploadRequest
	var fp *os.File
	for {
		fileData, err = stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { // first chunk contains file name
			tmpFile, fp, parent, uploadedModel, modelDefinitionID, err = saveFile(fileData)
			if err != nil {
				return "", "", &datamodel.Model{}, "", err
			}

			firstChunk = false
		}
		err = WriteToFp(fp, fileData.Content)
		if err != nil {
			return "", "", &datamodel.Model{}, "", err
		}
	}
	return tmpFile, parent, uploadedModel, modelDefinitionID, nil
}

func SaveOrganizationFile(stream modelPB.ModelPublicService_CreateOrganizationModelBinaryFileUploadServer) (tmpFile string, parent string, uploadedModel *datamodel.Model, modelDefinitionID string, err error) {
	firstChunk := true
	var fp *os.File
	var fileData *modelPB.CreateOrganizationModelBinaryFileUploadRequest
	for {
		fileData, err = stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", "", &datamodel.Model{}, "", fmt.Errorf("failed unexpectedly while reading chunks from stream")
		}

		if firstChunk { // first chunk contains file name
			tmpFile, fp, parent, uploadedModel, modelDefinitionID, err = saveFile(fileData)
			if err != nil {
				return "", "", &datamodel.Model{}, "", err
			}

			firstChunk = false
		}
		err = WriteToFp(fp, fileData.Content)
		if err != nil {
			return "", "", &datamodel.Model{}, "", err
		}
	}
	return tmpFile, parent, uploadedModel, modelDefinitionID, nil
}

func saveFile(fileData CreateNamespaceModelBinaryFileUploadRequestInterface) (tmpFile string, fp *os.File, parent string, uploadedModel *datamodel.Model, modelDefinitionID string, err error) {
	parent = fileData.GetParent()
	pbModel := fileData.GetModel()

	if pbModel == nil {
		return "", nil, "", &datamodel.Model{}, "", fmt.Errorf("failed to get model")
	}

	if parent == "" {
		return "", nil, "", &datamodel.Model{}, "", fmt.Errorf("failed to get namespace")
	}

	rdid, _ := uuid.NewV4()
	tmpFile = path.Join("/tmp", rdid.String()+".zip")
	fp, _ = os.Create(tmpFile)
	visibility := modelPB.Model_VISIBILITY_PRIVATE
	if pbModel.Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	}
	var description = ""
	if pbModel.Description != nil {
		description = *pbModel.Description
	}
	modelDefName := pbModel.ModelDefinition
	modelDefinitionID, err = resource.GetDefinitionID(modelDefName)
	if err != nil {
		return "", nil, "", &datamodel.Model{}, "", err
	}
	uploadedModel = &datamodel.Model{
		ID:         pbModel.Id,
		Visibility: datamodel.ModelVisibility(visibility),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		State:         datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Configuration: datatypes.JSON{},
	}
	if err != nil {
		return "", nil, "", &datamodel.Model{}, "", err
	}
	defer fp.Close()

	return tmpFile, fp, parent, uploadedModel, modelDefinitionID, nil
}

// GetJSON fetches the contents of the given URL
// and decodes it as JSON into the given result,
// which should be a pointer to the expected data.
func GetJSON(url string, result any) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http.Do with  MethodGet %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Do with  MethodGet status: %s", resp.Status)
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

func copyRayProto(dstPath string) error {
	files, err := filepath.Glob(fmt.Sprintf("%s/*pb2*", constant.RayProtoPath))
	if err != nil {
		return err
	}

	for _, filename := range files {
		sourceFileStat, err := os.Stat(filename)
		if err != nil {
			return err
		}

		if !sourceFileStat.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", filename)
		}

		source, err := os.Open(filename)
		if err != nil {
			return err
		}

		reader := NewProgressReader(source, sourceFileStat.Name())

		destination, err := os.Create(fmt.Sprintf("%s/%s", dstPath, sourceFileStat.Name()))
		if err != nil {
			return err
		}

		if _, err = io.Copy(destination, reader); err != nil {
			return err
		}

		source.Close()
		destination.Close()
	}

	return nil
}
