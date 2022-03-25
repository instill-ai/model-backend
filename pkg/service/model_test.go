package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/golang/mock/gomock"
	"github.com/instill-ai/model-backend/internal/inferenceserver"
	modelDB "github.com/instill-ai/model-backend/pkg/datamodel"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const NAMESPACE = "local-user"

func TestModelService_CreateModel(t *testing.T) {
	t.Run("CreateModel", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := modelDB.Model{
			Name:      "normalname",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockModelRepository := NewMockModelRepository(ctrl)
		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(modelDB.Model{}, nil).
			Times(2)
		mockModelRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)

		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		_, err := modelService.CreateModel(&newModel)
		assert.NoError(t, err)
	})
}

func TestModelService_CreateModel_InvalidName(t *testing.T) {
	t.Run("CreateModel_InvalidName", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := modelDB.Model{
			Name:      "#$%^",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		_, err := modelService.CreateModel(&newModel)
		if assert.Error(t, err) {
			assert.Equal(t, err, status.Error(codes.FailedPrecondition, "The name of model is invalid"))
		}

	})
}

func TestModelService_GetModelByName(t *testing.T) {
	t.Run("GetModelByName", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := modelDB.Model{
			Name:      "normalname",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockModelRepository := NewMockModelRepository(ctrl)
		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(modelDB.Model{}, nil).
			Times(1)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		_, err := modelService.GetModelByName(NAMESPACE, newModel.Name)
		assert.NoError(t, err)
	})
}

func TestModelService_CreateVersion(t *testing.T) {
	t.Run("CreateVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}
		modelVersion := modelDB.Version{
			ModelId:     1,
			Version:     1,
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockModelRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelDB.Version{}, nil)

		_, err := modelService.CreateVersion(modelVersion)
		assert.NoError(t, err)
	})
}

func TestModelService_GetModelVersion(t *testing.T) {
	t.Run("GetModelVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelDB.Version{}, nil)

		_, err := modelService.GetModelVersion(1, 1)
		assert.NoError(t, err)
	})
}

func TestModelService_GetModelVersions(t *testing.T) {
	t.Run("GetModelVersions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		mockModelRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]modelDB.Version{}, nil)

		_, err := modelService.GetModelVersions(1)
		assert.NoError(t, err)
	})
}

func TestModelService_GetModelVersionLatest(t *testing.T) {
	t.Run("GetModelVersionLatest", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		mockModelRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(modelDB.Version{}, nil)

		_, err := modelService.GetModelVersionLatest(1)
		assert.NoError(t, err)
	})
}

func TestModelService_GetFullModelData(t *testing.T) {
	t.Run("GetFullModelData", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}
		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(modelDB.Model{}, nil).
			Times(2)
		mockModelRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)
		_, _ = modelService.CreateModel(&newModel)

		modelVersion := modelDB.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		newModel.Versions = append(newModel.Versions, modelVersion)

		mockModelRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil)
		_, _ = modelService.CreateVersion(modelVersion)

		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockModelRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]modelDB.Version{}, nil)
		mockModelRepository.
			EXPECT().
			GetTModels(uint64(1)).
			Return([]modelDB.TModel{}, nil)

		_, err := modelService.GetFullModelData(NAMESPACE, "test")
		assert.NoError(t, err)
	})
}

func TestModelService_ModelInfer(t *testing.T) {
	t.Run("ModelInfer", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		triton := NewMockTritonService(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
			triton:          triton,
		}
		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(modelDB.Model{}, nil).
			Times(2)
		mockModelRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)
		_, _ = modelService.CreateModel(&newModel)

		modelVersion := modelDB.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		newModel.Versions = append(newModel.Versions, modelVersion)

		mockModelRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil)
		_, _ = modelService.CreateVersion(modelVersion)

		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		ensembleModel := modelDB.TModel{
			Name:    "essembleModel",
			Version: 1,
		}
		mockModelRepository.
			EXPECT().
			GetTEnsembleModel(uint64(1), uint64(1)).
			Return(ensembleModel, nil)

		modelConfigResponse := &inferenceserver.ModelConfigResponse{}
		modelMetadataResponse := &inferenceserver.ModelMetadataResponse{}
		modelInferResponse := &inferenceserver.ModelInferResponse{}
		postResponse := []string{"1.0:dog:1"}
		triton.
			EXPECT().
			ModelMetadataRequest(ensembleModel.Name, fmt.Sprint(ensembleModel.Version)).
			Return(modelMetadataResponse)
		triton.
			EXPECT().
			ModelConfigRequest(ensembleModel.Name, fmt.Sprint(ensembleModel.Version)).
			Return(modelConfigResponse)
		triton.
			EXPECT().
			ModelInferRequest(modelPB.Model_TASK_CLASSIFICATION, [][]byte{}, ensembleModel.Name, fmt.Sprint(ensembleModel.Version), modelMetadataResponse, modelConfigResponse).
			Return(modelInferResponse, nil)
		triton.
			EXPECT().
			PostProcess(modelInferResponse, modelMetadataResponse, modelPB.Model_TASK_CLASSIFICATION).
			Return(postResponse, nil)

		_, err := modelService.ModelInfer(NAMESPACE, "test", uint64(1), [][]byte{}, modelPB.Model_TASK_CLASSIFICATION)
		assert.NoError(t, err)
	})
}

func TestModelService_CreateModelBinaryFileUpload(t *testing.T) {
	t.Run("CreateModelBinaryFileUpload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		triton := NewMockTritonService(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
			triton:          triton,
		}
		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{},
		}
		modelVersion := modelDB.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockModelRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(modelDB.Version{}, fmt.Errorf("non-existed"))

		mockModelRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil).
			Times(2)
		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil).
			Times(2)
		versionInDB, _ := modelService.CreateVersion(modelVersion)

		mockModelRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]modelDB.Version{modelVersion}, nil).Times(2)
		_, _ = modelService.GetModelVersions(uint64(1))

		uploadModel := modelDB.Model{
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{versionInDB},
		}
		_, err := modelService.CreateModelBinaryFileUpload(NAMESPACE, &uploadModel)
		assert.NoError(t, err)
	})
}

func TestModelService_HandleCreateModelMultiFormDataUpload(t *testing.T) {
	t.Run("HandleCreateModelMultiFormDataUpload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		triton := NewMockTritonService(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
			triton:          triton,
		}
		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{},
		}
		modelVersion := modelDB.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockModelRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockModelRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(modelDB.Version{}, fmt.Errorf("non-existed"))

		mockModelRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil).
			Times(2)
		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil).
			Times(2)
		versionInDB, _ := modelService.CreateVersion(modelVersion)

		mockModelRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]modelDB.Version{modelVersion}, nil).Times(2)
		_, _ = modelService.GetModelVersions(uint64(1))

		uploadModel := modelDB.Model{
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{versionInDB},
		}
		_, err := modelService.CreateModelBinaryFileUpload(NAMESPACE, &uploadModel)
		assert.NoError(t, err)
	})
}

func TestModelService_ListModels(t *testing.T) {
	t.Run("ListModels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		mockModelRepository.
			EXPECT().
			ListModels(modelDB.ListModelQuery{Namespace: NAMESPACE}).
			Return([]modelDB.Model{}, nil)

		_, err := modelService.ListModels(NAMESPACE)
		assert.NoError(t, err)
	})
}

func TestModelService_UpdateModelVersion(t *testing.T) {
	t.Run("UpdateModelVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{},
		}
		mockModelRepository.
			EXPECT().
			GetModelByName(NAMESPACE, newModel.Name).
			Return(newModel, nil).
			Times(2)
		_, _ = modelService.GetModelByName(NAMESPACE, newModel.Name)

		mockModelRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelDB.Version{}, nil).
			Times(2)
		_, _ = modelService.GetModelVersion(uint64(1), uint64(1))

		_, err := modelService.UpdateModelVersion(NAMESPACE, &modelPB.UpdateModelVersionRequest{
			Name:    newModel.Name,
			Version: uint64(1),
			VersionPatch: &modelPB.UpdateModelVersionPatch{
				Description: "updated description",
			},
		})
		assert.NoError(t, err)
	})
}

func TestModelService_DeleteModel(t *testing.T) {
	t.Run("DeleteModel", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockModelRepository := NewMockModelRepository(ctrl)
		modelService := modelService{
			modelRepository: mockModelRepository,
		}

		newModel := modelDB.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []modelDB.Version{},
		}
		mockModelRepository.
			EXPECT().
			GetModelByName(NAMESPACE, newModel.Name).
			Return(newModel, nil).
			Times(2)
		_, _ = modelService.GetModelByName(NAMESPACE, newModel.Name)

		mockModelRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]modelDB.Version{}, nil).Times(2)
		mockModelRepository.
			EXPECT().
			GetTModels(uint64(1)).
			Return([]modelDB.TModel{}, nil).Times(1)
		mockModelRepository.
			EXPECT().
			DeleteModel(uint64(1)).
			Return(nil).Times(1)
		_, _ = modelService.GetModelVersions(uint64(1))

		err := modelService.DeleteModel(NAMESPACE, newModel.Name)
		assert.NoError(t, err)
	})
}
