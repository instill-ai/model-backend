package service_test

//go:generate mockgen -destination mock_triton_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/internal/triton Triton
//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/repository Repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/model-backend/internal/inferenceserver"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

const NAMESPACE = "local-user"

func TestCreateModel(t *testing.T) {
	t.Run("CreateModel", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := datamodel.Model{
			Name:      "normalname",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(datamodel.Model{}, nil).
			Times(2)
		mockRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)

		s := service.NewService(mockRepository, nil)

		_, err := s.CreateModel(&newModel)
		assert.NoError(t, err)
	})
}

func TestCreateModel_InvalidName(t *testing.T) {
	t.Run("CreateModel_InvalidName", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := datamodel.Model{
			Name:      "#$%^",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		_, err := s.CreateModel(&newModel)
		if assert.Error(t, err) {
			assert.Equal(t, err, status.Error(codes.FailedPrecondition, "The name of model is invalid"))
		}

	})
}

func TestGetModelByName(t *testing.T) {
	t.Run("GetModelByName", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		newModel := datamodel.Model{
			Name:      "normalname",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(datamodel.Model{}, nil).
			Times(1)
		s := service.NewService(mockRepository, nil)

		_, err := s.GetModelByName(NAMESPACE, newModel.Name)
		assert.NoError(t, err)
	})
}

func TestCreateVersion(t *testing.T) {
	t.Run("CreateVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		modelVersion := datamodel.Version{
			ModelId:     1,
			Version:     1,
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(datamodel.Version{}, nil)

		_, err := s.CreateVersion(modelVersion)
		assert.NoError(t, err)
	})
}

func TestGetModelVersion(t *testing.T) {
	t.Run("GetModelVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(datamodel.Version{}, nil)

		_, err := s.GetModelVersion(1, 1)
		assert.NoError(t, err)
	})
}

func TestGetModelVersions(t *testing.T) {
	t.Run("GetModelVersions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		mockRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]datamodel.Version{}, nil)

		_, err := s.GetModelVersions(1)
		assert.NoError(t, err)
	})
}

func TestGetModelVersionLatest(t *testing.T) {
	t.Run("GetModelVersionLatest", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		mockRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(datamodel.Version{}, nil)

		_, err := s.GetModelVersionLatest(1)
		assert.NoError(t, err)
	})
}

func TestGetFullModelData(t *testing.T) {
	t.Run("GetFullModelData", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(datamodel.Model{}, nil).
			Times(2)
		mockRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)
		_, _ = s.CreateModel(&newModel)

		modelVersion := datamodel.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		newModel.Versions = append(newModel.Versions, modelVersion)

		mockRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil)
		_, _ = s.CreateVersion(modelVersion)

		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]datamodel.Version{}, nil)
		mockRepository.
			EXPECT().
			GetTModels(uint64(1)).
			Return([]datamodel.TModel{}, nil)

		_, err := s.GetFullModelData(NAMESPACE, "test")
		assert.NoError(t, err)
	})
}

func TestModelInfer(t *testing.T) {
	t.Run("ModelInfer", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		triton := NewMockTriton(ctrl)
		s := service.NewService(mockRepository, triton)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
		}
		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(datamodel.Model{}, nil).
			Times(2)
		mockRepository.
			EXPECT().
			CreateModel(newModel).
			Return(nil)
		_, _ = s.CreateModel(&newModel)

		modelVersion := datamodel.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		newModel.Versions = append(newModel.Versions, modelVersion)

		mockRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil)
		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil)
		_, _ = s.CreateVersion(modelVersion)

		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		ensembleModel := datamodel.TModel{
			Name:    "essembleModel",
			Version: 1,
		}
		mockRepository.
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

		_, err := s.ModelInfer(NAMESPACE, "test", uint64(1), [][]byte{}, modelPB.Model_TASK_CLASSIFICATION)
		assert.NoError(t, err)
	})
}

func TestCreateModelBinaryFileUpload(t *testing.T) {
	t.Run("CreateModelBinaryFileUpload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		triton := NewMockTriton(ctrl)
		s := service.NewService(mockRepository, triton)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{},
		}
		modelVersion := datamodel.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(datamodel.Version{}, fmt.Errorf("non-existed"))

		mockRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil).
			Times(2)
		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil).
			Times(2)
		versionInDB, _ := s.CreateVersion(modelVersion)

		mockRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]datamodel.Version{modelVersion}, nil).Times(2)
		_, _ = s.GetModelVersions(uint64(1))

		uploadModel := datamodel.Model{
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{versionInDB},
		}
		_, err := s.CreateModelBinaryFileUpload(NAMESPACE, &uploadModel)
		assert.NoError(t, err)
	})
}

func TestHandleCreateModelMultiFormDataUpload(t *testing.T) {
	t.Run("HandleCreateModelMultiFormDataUpload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		triton := NewMockTriton(ctrl)
		s := service.NewService(mockRepository, triton)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{},
		}
		modelVersion := datamodel.Version{
			ModelId:     uint64(1),
			Version:     uint64(1),
			Description: "This is version 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepository.
			EXPECT().
			GetModelByName(gomock.Eq(NAMESPACE), gomock.Eq(newModel.Name)).
			Return(newModel, nil).
			Times(1)
		mockRepository.
			EXPECT().
			GetModelVersionLatest(uint64(1)).
			Return(datamodel.Version{}, fmt.Errorf("non-existed"))

		mockRepository.
			EXPECT().
			CreateVersion(modelVersion).
			Return(nil).
			Times(2)
		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(modelVersion, nil).
			Times(2)
		versionInDB, _ := s.CreateVersion(modelVersion)

		mockRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]datamodel.Version{modelVersion}, nil).Times(2)
		_, _ = s.GetModelVersions(uint64(1))

		uploadModel := datamodel.Model{
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{versionInDB},
		}
		_, err := s.CreateModelBinaryFileUpload(NAMESPACE, &uploadModel)
		assert.NoError(t, err)
	})
}

func TestListModels(t *testing.T) {
	t.Run("ListModels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		mockRepository.
			EXPECT().
			ListModels(datamodel.ListModelQuery{Namespace: NAMESPACE}).
			Return([]datamodel.Model{}, nil)

		_, err := s.ListModels(NAMESPACE)
		assert.NoError(t, err)
	})
}

func TestUpdateModelVersion(t *testing.T) {
	t.Run("UpdateModelVersion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{},
		}
		mockRepository.
			EXPECT().
			GetModelByName(NAMESPACE, newModel.Name).
			Return(newModel, nil).
			Times(2)
		_, _ = s.GetModelByName(NAMESPACE, newModel.Name)

		mockRepository.
			EXPECT().
			GetModelVersion(uint64(1), uint64(1)).
			Return(datamodel.Version{}, nil).
			Times(2)
		_, _ = s.GetModelVersion(uint64(1), uint64(1))

		_, err := s.UpdateModelVersion(NAMESPACE, &modelPB.UpdateModelVersionRequest{
			Name:    newModel.Name,
			Version: uint64(1),
			VersionPatch: &modelPB.UpdateModelVersionPatch{
				Description: "updated description",
			},
		})
		assert.NoError(t, err)
	})
}

func TestDeleteModel(t *testing.T) {
	t.Run("DeleteModel", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepository := NewMockRepository(ctrl)
		s := service.NewService(mockRepository, nil)

		newModel := datamodel.Model{
			Id:        1,
			Name:      "test",
			Task:      uint64(modelPB.Model_TASK_CLASSIFICATION),
			Namespace: NAMESPACE,
			Versions:  []datamodel.Version{},
		}
		mockRepository.
			EXPECT().
			GetModelByName(NAMESPACE, newModel.Name).
			Return(newModel, nil).
			Times(2)
		_, _ = s.GetModelByName(NAMESPACE, newModel.Name)

		mockRepository.
			EXPECT().
			GetModelVersions(uint64(1)).
			Return([]datamodel.Version{}, nil).Times(2)
		mockRepository.
			EXPECT().
			GetTModels(uint64(1)).
			Return([]datamodel.TModel{}, nil).Times(1)
		mockRepository.
			EXPECT().
			DeleteModel(uint64(1)).
			Return(nil).Times(1)
		_, _ = s.GetModelVersions(uint64(1))

		err := s.DeleteModel(NAMESPACE, newModel.Name)
		assert.NoError(t, err)
	})
}
