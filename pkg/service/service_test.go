package service_test

//go:generate mockgen -destination mock_triton_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/triton Triton
//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/repository Repository

import (
	"context"
	// "database/sql"
	// "fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"github.com/instill-ai/model-backend/pkg/service"

	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	// inferenceserver "github.com/instill-ai/model-backend/pkg/triton/inferenceserver"
	// commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

const ID = "modelID"
const OWNER = "users/909c3278-f7d1-461c-9352-87741bef1ds1"

var MODEL_DEFINITION, _ = uuid.FromString("909c3278-f7d1-461c-9352-87741bef11d3")

//TODO: async method, need to figure out how to test this
// func TestCreateModel(t *testing.T) {
// 	t.Run("CreateModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uuid.UUID{}},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(datamodel.Model{}, nil).
// 			Times(2)
// 		mockRepository.
// 			EXPECT().
// 			CreateModel(newModel).
// 			Return(nil)

// 		s := service.NewService(mockRepository, nil, nil, nil, nil)

// 		_, err := s.CreateModelAsync(OWNER, &newModel)
// 		assert.NoError(t, err)
// 	})
// }

// func TestGetModelByID(t *testing.T) {
// 	t.Run("TestGetModelByID", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uuid.UUID{}},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(datamodel.Model{}, nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.GetModelByID(context.Background(), OWNER, newModel.ID, modelPB.View_VIEW_FULL)
// 		assert.NoError(t, err)
// 	})
// }

// func TestGetModelByUID(t *testing.T) {
// 	t.Run("TestGetModelByUID", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByUID(gomock.Eq(OWNER), gomock.Eq(newModel.UID), modelPB.View_VIEW_FULL).
// 			Return(datamodel.Model{}, nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.GetModelByUID(context.Background(), OWNER, uid, modelPB.View_VIEW_FULL)
// 		assert.NoError(t, err)
// 	})
// }

// func TestDeleteModel(t *testing.T) {
// 	t.Run("TestDeleteModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic:        datamodel.BaseDynamic{UID: uid},
// 			ID:                 ID,
// 			Description:        "this is a test model",
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(datamodel.Model{}, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetModelInstances(gomock.Eq(newModel.UID)).
// 			Return([]datamodel.ModelInstance{}, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			DeleteModel(gomock.Eq(newModel.UID)).
// 			Return(nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil)

// 		err := s.DeleteModel(OWNER, ID)
// 		assert.NoError(t, err)
// 	})
// }

// func TestRenameModel(t *testing.T) {
// 	t.Run("TestRenameModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}

// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			UpdateModel(newModel.UID, datamodel.Model{
// 				ID: "new ID",
// 			}).
// 			Return(nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), "new ID", modelPB.View_VIEW_FULL).
// 			Return(datamodel.Model{
// 				BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 				ID:          "new ID",
// 				Description: sql.NullString{
// 					String: "this is a test model",
// 					Valid:  true,
// 				},
// 				ModelDefinitionUid: MODEL_DEFINITION,
// 				Owner:              OWNER,
// 			}, nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.RenameModel(context.Background(), OWNER, ID, "new ID")
// 		assert.NoError(t, err)
// 	})
// }

// func TestPublishModel(t *testing.T) {
// 	t.Run("TestPublishModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			UpdateModel(newModel.UID, datamodel.Model{
// 				ID:         ID,
// 				Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC),
// 			}).
// 			Return(nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.PublishModel(context.Background(), OWNER, ID)
// 		assert.NoError(t, err)
// 	})
// }

// func TestUnpublishModel(t *testing.T) {
// 	t.Run("TestUnpublishModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			UpdateModel(newModel.UID, datamodel.Model{
// 				ID:         ID,
// 				Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE),
// 			}).
// 			Return(nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.UnpublishModel(context.Background(), OWNER, ID)
// 		assert.NoError(t, err)
// 	})
// }

// func TestUpdateModel(t *testing.T) {
// 	t.Run("TestUpdateModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		uid := uuid.UUID{}
// 		newModel := datamodel.Model{
// 			BaseDynamic: datamodel.BaseDynamic{UID: uid},
// 			ID:          ID,
// 			Description: sql.NullString{
// 				String: "this is a test model",
// 				Valid:  true,
// 			},
// 			ModelDefinitionUid: MODEL_DEFINITION,
// 			Owner:              OWNER,
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			UpdateModel(newModel.UID, newModel).
// 			Return(nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetModelByID(gomock.Eq(OWNER), gomock.Eq(newModel.ID), modelPB.View_VIEW_FULL).
// 			Return(newModel, nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, err := s.UpdateModel(context.Background(), newModel.UID, &newModel)
// 		assert.NoError(t, err)
// 	})
// }

// func TestListModell(t *testing.T) {
// 	t.Run("TestListModel", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			ListModels(OWNER, modelPB.View_VIEW_FULL, int(100), "").
// 			Return([]datamodel.Model{}, "", int64(100), nil).
// 			Times(1)
// 		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

// 		_, _, _, err := s.ListModels(context.Background(), OWNER, modelPB.View_VIEW_FULL, 100, "")
// 		assert.NoError(t, err)
// 	})
// }

// func TestModelInfer(t *testing.T) {
// 	t.Run("ModelInfer", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockRepository := NewMockRepository(ctrl)
// 		triton := NewMockTriton(ctrl)
// 		s := service.NewService(mockRepository, triton, nil, nil, nil, nil)

// 		uid := uuid.UUID{}

// 		ensembleModel := datamodel.TritonModel{
// 			Name:    "ensembleModel",
// 			Version: 1,
// 		}
// 		mockRepository.
// 			EXPECT().
// 			GetTritonEnsembleModel(uid).
// 			Return(ensembleModel, nil)

// 		modelConfigResponse := &inferenceserver.ModelConfigResponse{}
// 		modelMetadataResponse := &inferenceserver.ModelMetadataResponse{}
// 		modelInferResponse := &inferenceserver.ModelInferResponse{}
// 		postResponse := []string{"1.0:dog:1"}
// 		triton.
// 			EXPECT().
// 			ModelMetadataRequest(context.Background(), ensembleModel.Name, fmt.Sprint(ensembleModel.Version)).
// 			Return(modelMetadataResponse)
// 		triton.
// 			EXPECT().
// 			ModelConfigRequest(context.Background(), ensembleModel.Name, fmt.Sprint(ensembleModel.Version)).
// 			Return(modelConfigResponse)
// 		triton.
// 			EXPECT().
// 			ModelInferRequest(context.Background(), commonPB.Task_TASK_CLASSIFICATION, [][]byte{}, ensembleModel.Name, fmt.Sprint(ensembleModel.Version), modelMetadataResponse, modelConfigResponse).
// 			Return(modelInferResponse, nil)
// 		triton.
// 			EXPECT().
// 			PostProcess(modelInferResponse, modelMetadataResponse, commonPB.Task_TASK_CLASSIFICATION).
// 			Return(postResponse, nil)

// 		_, err := s.ModelInfer(context.Background(), uid, [][]byte{}, commonPB.Task_TASK_CLASSIFICATION)
// 		assert.NoError(t, err)
// 	})
// }

// func TestDeployModelInstance(t *testing.T) {
// 	t.Run("TestDeployModelInstance", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockRepository := NewMockRepository(ctrl)
// 		triton := NewMockTriton(ctrl)
// 		s := service.NewService(mockRepository, triton, nil, nil, nil)

// 		uid := uuid.UUID{}

// 		ensembleModel := datamodel.TritonModel{
// 			Name:    "ensembleModel",
// 			Version: 1,
// 		}
// 		mockRepository.
// 			EXPECT().
// 			GetTritonEnsembleModel(uid).
// 			Return(ensembleModel, nil)

// 		triton.
// 			EXPECT().
// 			LoadModelRequest(ensembleModel.Name).
// 			Return(nil, nil)

// 		mockRepository.
// 			EXPECT().
// 			UpdateModelInstance(uid, datamodel.ModelInstance{
// 				State: datamodel.ModelInstanceState(modelPB.Model_STATE_ONLINE),
// 			}).
// 			Return(nil)

// 		err := s.DeployModelInstanceAsync(uid)
// 		assert.NoError(t, err)
// 	})
// }

// func TestUndeployModelInstance(t *testing.T) {
// 	t.Run("TestUndeployModelInstance", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockRepository := NewMockRepository(ctrl)
// 		triton := NewMockTriton(ctrl)
// 		s := service.NewService(mockRepository, triton, nil, nil, nil)

// 		uid := uuid.UUID{}

// 		ensembleModel := datamodel.TritonModel{
// 			Name:    "ensembleModel",
// 			Version: 1,
// 		}
// 		mockRepository.
// 			EXPECT().
// 			GetTritonModels(uid).
// 			Return([]datamodel.TritonModel{ensembleModel}, nil)

// 		triton.
// 			EXPECT().
// 			UnloadModelRequest(ensembleModel.Name).
// 			Return(nil, nil)

// 		mockRepository.
// 			EXPECT().
// 			UpdateModelInstance(uid, datamodel.ModelInstance{
// 				State: datamodel.ModelInstanceState(modelPB.Model_STATE_OFFLINE),
// 			}).
// 			Return(nil)

// 		err := s.UndeployModelInstanceAsync(uid)
// 		assert.NoError(t, err)
// 	})
// }

func TestGetModelDefinition(t *testing.T) {
	t.Run("TestGetModelDefinition", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetModelDefinition("github").
			Return(&datamodel.ModelDefinition{}, nil).
			Times(1)
		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

		_, err := s.GetModelDefinition(context.Background(), "github")
		assert.NoError(t, err)
	})
}

func TestListModelDefinitions(t *testing.T) {
	t.Run("TestListModelDefinition", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			ListModelDefinitions(modelPB.View_VIEW_FULL, int(100), "").
			Return([]*datamodel.ModelDefinition{}, "", int64(100), nil).
			Times(1)
		s := service.NewService(mockRepository, nil, nil, nil, nil, nil)

		_, _, _, err := s.ListModelDefinitions(context.Background(), modelPB.View_VIEW_FULL, 100, "")
		assert.NoError(t, err)
	})
}
