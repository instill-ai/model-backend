package rpc

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/instill-ai/model-backend/configs"
	database "github.com/instill-ai/model-backend/internal/db"
	metadataUtil "github.com/instill-ai/model-backend/internal/grpc/metadata"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/models"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/services"
	"github.com/instill-ai/protogen-go/model"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type serviceHandlers struct {
	modelService services.ModelService
}

func NewServiceHandlers(modelService services.ModelService) model.ModelServer {
	return &serviceHandlers{
		modelService: modelService,
	}
}

//writeToFp takes in a file pointer and byte array and writes the byte array into the file
//returns error if pointer is nil or error in writing to file
func writeToFp(fp *os.File, data []byte) error {
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

func unzip(filePath string, dstDir string) ([]*models.Model, []*models.Version, bool) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Println("Error when open zip file ", err)
		return []*models.Model{}, []*models.Version{}, false
	}
	defer archive.Close()

	var createdModels []*models.Model
	var createdVersions []*models.Version
	var currentModelName string
	for _, f := range archive.File {
		if strings.Contains(f.Name, "__MACOSX") { // ignore temp directory in macos
			continue
		}
		filePath := filepath.Join(dstDir, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return []*models.Model{}, []*models.Version{}, false
		}
		if f.FileInfo().IsDir() {
			dirName := f.Name
			if string(dirName[len(dirName)-1]) == "/" {
				dirName = dirName[:len(dirName)-1]
			}
			if !strings.Contains(dirName, "/") { // top directory model
				currentModelName = dirName
				newModel := &models.Model{
					Name:       dirName,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
					Type:       "tensorrt",
					Framework:  "pytorch",
					Optimized:  false,
					Icon:       "",
					Visibility: "public",
				}
				createdModels = append(createdModels, newModel)
			} else { // version folder
				patternVersionFolder := fmt.Sprintf("^%v/[0-9]+$", currentModelName)
				match, _ := regexp.MatchString(patternVersionFolder, dirName)
				if match {
					elems := strings.Split(dirName, "/")
					sVersion := elems[len(elems)-1]
					iVersion, err := strconv.Atoi(sVersion)
					if err == nil {
						newVersion := &models.Version{
							Version:   int32(iVersion),
							CreatedAt: time.Now(),
							ModelName: currentModelName,
							UpdatedAt: time.Now(),
							Status:    "offline",
							Metadata:  models.JSONB{},
						}
						createdVersions = append(createdVersions, newVersion)
					}

				}
			}

			fmt.Println("creating directory... ", filePath)
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return []*models.Model{}, []*models.Version{}, false
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return []*models.Model{}, []*models.Version{}, false
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return []*models.Model{}, []*models.Version{}, false
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return []*models.Model{}, []*models.Version{}, false
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return createdModels, createdVersions, true
}

func saveFile(stream model.Model_CreateModelByUploadServer) (outFile string, err error) {
	firstChunk := true
	var fp *os.File

	var fileData *model.CreateModelRequest

	var tmpFile string

	for {
		fileData, err = stream.Recv() //ignoring the data  TO-Do save files received
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed unexpectadely while reading chunks from stream")
			return "", err
		}

		if firstChunk { //first chunk contains file name
			tmpFile = path.Join("/tmp", uuid.New().String()+".zip")
			fp, err = os.Create(tmpFile)
			if err != nil {
				return "", err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = writeToFp(fp, fileData.Content)
		if err != nil {
			return "", err
		}
	}
	return tmpFile, nil
}

func savePredictInput(stream model.Model_PredictModelByUploadServer) (imageFile string, modelId string, version int32, modelType triton.CVTask, err error) {
	firstChunk := true
	var fp *os.File

	var fileData *model.PredictModelRequest

	var tmpFile string

	for {
		fileData, err = stream.Recv() //ignoring the data  TO-Do save files received
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed unexpectadely while reading chunks from stream")
			return "", "", -1, 0, err
		}

		if firstChunk { //first chunk contains file name
			modelId = fileData.Name
			version = fileData.Version
			modelType = triton.CVTask(fileData.Type)

			tmpFile = path.Join("/tmp/", uuid.New().String())
			fp, err = os.Create(tmpFile)
			if err != nil {
				return "", "", -1, 0, err
			}
			defer fp.Close()

			firstChunk = false
		}
		err = writeToFp(fp, fileData.Content)
		if err != nil {
			return "", "", -1, 0, err
		}
	}
	return tmpFile, modelId, version, modelType, nil
}

func makeError(statusCode codes.Code, title string, detail string) error {
	err := &models.Error{
		Title:  title,
		Detail: detail,
	}
	data, _ := json.Marshal(err)
	return status.Error(statusCode, string(data))
}

func makeResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(models.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

func getUsername(ctx context.Context) (string, error) {
	if metadatas, ok := metadataUtil.ExtractFromMetadata(ctx, "Username"); ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "Username not found in your request")
		}
		return metadatas[0], nil
	} else {
		return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
	}
}

func (s *serviceHandlers) Liveness(ctx context.Context, pb *emptypb.Empty) (*model.HealthCheckResponse, error) {
	if !triton.IsTritonServerReady() {
		return &model.HealthCheckResponse{Status: 503}, nil
	}

	return &model.HealthCheckResponse{Status: 200}, nil
}

func (s *serviceHandlers) Readiness(ctx context.Context, pb *emptypb.Empty) (*model.HealthCheckResponse, error) {
	if !triton.IsTritonServerReady() {
		return &model.HealthCheckResponse{Status: 503}, nil
	}

	return &model.HealthCheckResponse{Status: 200}, nil
}

func HandleCreateModelByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		if username == "" {
			makeResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
		}

		db := database.GetConnection()
		modelRepository := repository.NewModelRepository(db)
		modelService := services.NewModelService(modelRepository)

		err := r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeResponse(w, 500, "Internal Error", "Error while reading file from request")
		}

		file, _, err := r.FormFile("content")
		if err != nil {
			makeResponse(w, 500, "Internal Error", "Error while reading file from request")
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		buf := bytes.NewBuffer(make([]byte, 0))
		part := make([]byte, 1024)
		count := 0
		for {
			if count, err = reader.Read(part); err != nil {
				break
			}
			buf.Write(part[:count])
		}
		if err != io.EOF {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		tmpFile := path.Join("/tmp", uuid.New().String())
		fp, err := os.Create(tmpFile)
		if err != nil {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		err = writeToFp(fp, buf.Bytes())
		if err != nil {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		createdModels, createdVersions, isOk := unzip(tmpFile, configs.Config.TritonServer.ModelStore)
		if !isOk || len(createdModels) == 0 {
			makeResponse(w, 400, "Add Model Error", "Could not extract zip file")
		}

		respModels, err := modelService.HandleCreateModelByUpload(username, createdModels, createdVersions)
		if err != nil {
			makeResponse(w, 500, "Add Model Error", err.Error())
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		ret, _ := json.Marshal(respModels)
		_, _ = w.Write(ret)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

func (s *serviceHandlers) CreateModel(ctx context.Context, in *model.CreateModelRequest) (*model.CreateModelsResponse, error) {
	//TODO support url and base64 content
	return &model.CreateModelsResponse{}, nil
}

// AddModel - upload a model to the model server
func (s *serviceHandlers) CreateModelByUpload(stream model.Model_CreateModelByUploadServer) (err error) {
	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}

	tmpFile, err := saveFile(stream)
	if err != nil {
		return makeError(400, "Save File Error", err.Error())
	}

	// extract zip file from tmp to models directory
	createdModels, createdVersions, isOk := unzip(tmpFile, configs.Config.TritonServer.ModelStore)
	if !isOk || len(createdModels) == 0 {
		return makeError(400, "Save File Error", "Could not extract zip file")
	}

	respModels, err := s.modelService.CreateModelByUpload(username, createdModels, createdVersions)
	if err != nil {
		return makeError(500, "Add Model Error", err.Error())
	}

	var res model.CreateModelsResponse
	res.Models = respModels
	err = stream.SendAndClose(&res)
	if err != nil {
		return makeError(500, "Add Model Error", err.Error())
	}

	return
}

func (s *serviceHandlers) UpdateModel(ctx context.Context, in *model.UpdateModelRequest) (*model.ModelInfo, error) {
	if !triton.IsTritonServerReady() {
		return &model.ModelInfo{}, makeError(503, "LoadModel Error", "Triton Server not ready yet")
	}

	username, err := getUsername(ctx)
	if err != nil {
		return &model.ModelInfo{}, err
	}

	return s.modelService.UpdateModel(username, in)
}

func (s *serviceHandlers) ListModels(ctx context.Context, in *model.ListModelRequest) (*model.ListModelResponse, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &model.ListModelResponse{}, err
	}

	resModels, err := s.modelService.ListModels(username)
	return &model.ListModelResponse{Models: resModels}, err
}

func (s *serviceHandlers) PredictModel(ctx context.Context, in *model.PredictModelRequest) (*structpb.Struct, error) {
	fmt.Println("PredictModel", in)
	return &structpb.Struct{}, nil
}

func (s *serviceHandlers) PredictModelByUpload(stream model.Model_PredictModelByUploadServer) error {
	if !triton.IsTritonServerReady() {
		return makeError(503, "PredictModel", "Triton Server not ready yet")
	}

	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}

	imageFile, modelName, version, cvTask, err := savePredictInput(stream)

	if err != nil {
		return makeError(500, "PredictModel", "Could not save the file")
	}

	response, err := s.modelService.PredictModelByUpload(username, modelName, version, imageFile, cvTask)

	if err != nil {
		return err
	}

	var data = &structpb.Struct{}
	var b []byte
	switch cvTask {
	case triton.Classification:
		b, err = json.Marshal(response.(*model.ClassificationOutputs))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}

	case triton.Detection:
		b, err = json.Marshal(response.(*model.DetectionOutputs))
		if err != nil {
			return makeError(500, "PredictModel", err.Error())
		}
	}
	err = protojson.Unmarshal(b, data)
	if err != nil {
		return makeError(500, "PredictModel", err.Error())
	}
	err = stream.SendAndClose(data)
	return err
}

func HandlePredictModelByUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		modelName := pathParams["name"]

		if username == "" {
			makeResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
		}
		if modelName == "" {
			makeResponse(w, 422, "Required parameter missing", "Required parameter mode name not found")
		}

		modelVersion, err := strconv.Atoi(r.FormValue("version"))
		if err != nil {
			makeResponse(w, 400, "Wrong parameter type", "Version should be a number greater than 0")
		}
		modelType, err := strconv.Atoi(r.FormValue("type"))
		if err != nil {
			makeResponse(w, 400, "Wrong parameter type", "Type should be a number greater than or equal 0")
		}
		cvTask := triton.CVTask(modelType)

		db := database.GetConnection()
		modelRepository := repository.NewModelRepository(db)
		modelService := services.NewModelService(modelRepository)

		_, err = modelService.GetModelByName(username, modelName)
		if err != nil {
			makeResponse(w, 404, "Model not found", "The model not found in server")
		}

		err = r.ParseMultipartForm(4 << 20)
		if err != nil {
			makeResponse(w, 500, "Internal Error", "Error while reading file from request")
		}

		file, _, err := r.FormFile("content")
		if err != nil {
			makeResponse(w, 500, "Internal Error", "Error while reading file from request")
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		buf := bytes.NewBuffer(make([]byte, 0))
		part := make([]byte, 1024)
		count := 0
		for {
			if count, err = reader.Read(part); err != nil {
				break
			}
			buf.Write(part[:count])
		}
		if err != io.EOF {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		tmpFile := path.Join("/tmp", uuid.New().String()+".jpg")
		fp, err := os.Create(tmpFile)
		if err != nil {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		err = writeToFp(fp, buf.Bytes())
		if err != nil {
			makeResponse(w, 400, "Internal Error", "Error reading input file")
		}

		response, err := modelService.PredictModelByUpload(username, modelName, int32(modelVersion), tmpFile, cvTask)
		if err != nil {
			makeResponse(w, 500, "Error Predict Model", err.Error())
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		ret, _ := json.Marshal(response)
		_, _ = w.Write(ret)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}

func (s *serviceHandlers) GetModel(ctx context.Context, in *model.GetModelRequest) (*model.ModelInfo, error) {
	username, err := getUsername(ctx)
	if err != nil {
		return &model.ModelInfo{}, err
	}
	return s.modelService.GetModelMetaData(username, in.Name)
}

func (s *serviceHandlers) DeleteModel(ctx context.Context, in *model.DeleteModelRequest) (*emptypb.Empty, error) {
	//TODO support url and base64 content
	return &emptypb.Empty{}, makeError(500, "DeleteModel", "Not supported yet")
}
