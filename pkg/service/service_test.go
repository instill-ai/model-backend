package service_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"

	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/mock"
	"github.com/instill-ai/model-backend/pkg/service"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
	openfga "github.com/openfga/api/proto/openfga/v1"
)

const ID = "modelID"

const OWNER = "users/909c3278-f7d1-461c-9352-87741bef1ds1"

var ModelDefinition, _ = uuid.FromString("909c3278-f7d1-461c-9352-87741bef11d3")

// TODO: async method, need to figure out how to test this
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

func TestGetModelByUID(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Run("TestGetModelByUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		uid := uuid.UUID{}
		newModel := datamodel.Model{
			BaseDynamic: datamodel.BaseDynamic{UID: uid},
			ID:          ID,
			Description: sql.NullString{
				String: "this is a test model",
				Valid:  true,
			},
			ModelDefinitionUID: ModelDefinition,
			Owner:              OWNER,
			Task:               datamodel.ModelTask(commonpb.Task_TASK_CLASSIFICATION),
		}
		datamodel.TaskInputJSON = map[string]any{
			commonpb.Task_TASK_CLASSIFICATION.String(): map[string]any{},
		}
		datamodel.TaskOutputJSON = map[string]any{
			commonpb.Task_TASK_CLASSIFICATION.String(): map[string]any{},
		}

		ctx := context.Background()
		mockRepository := mock.NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetModelByUID(ctx, gomock.Eq(newModel.UID), false, false).
			Return(&newModel, nil).
			Times(1)
		mockRepository.EXPECT().GetModelDefinitionByUID(newModel.ModelDefinitionUID).Return(&datamodel.ModelDefinition{}, nil).Times(1)

		wc := mock.NewMockOpenFGAServiceClient(ctrl)
		wc.EXPECT().ListStores(gomock.Any(), gomock.Any()).Return(
			&openfga.ListStoresResponse{
				Stores: []*openfga.Store{
					{Id: "id"},
				},
			},
			nil).Times(1)
		wc.EXPECT().ReadAuthorizationModels(gomock.Any(), gomock.Any()).Return(
			&openfga.ReadAuthorizationModelsResponse{
				AuthorizationModels: []*openfga.AuthorizationModel{
					{Id: "id"},
				},
			},
			nil).Times(1)
		wc.EXPECT().Check(gomock.Any(), gomock.Any()).Return(&openfga.CheckResponse{Allowed: true}, nil).Times(3)
		aclClient := acl.NewACLClient(wc, nil, rc)

		mgmtPrivateService := mock.NewMockMgmtPrivateServiceClient(ctrl)
		mgmtPrivateService.EXPECT().LookUpUserAdmin(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&mgmtpb.LookUpUserAdminResponse{User: &mgmtpb.User{
				Id: "id",
			}},
			nil).Times(2)

		s := service.NewService(mockRepository, nil, mgmtPrivateService, nil, rc, nil, nil, &aclClient, "")

		_, err = s.GetModelByUID(ctx, uid, modelPB.View_VIEW_FULL)
		assert.NoError(t, err)
	})
}

func TestListModel(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Run("TestListModel", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockRepository := mock.NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			ListModels(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*datamodel.Model{}, int64(100), "", nil).
			Times(1)

		wc := mock.NewMockOpenFGAServiceClient(ctrl)
		wc.EXPECT().ListStores(gomock.Any(), gomock.Any()).Return(
			&openfga.ListStoresResponse{
				Stores: []*openfga.Store{
					{Id: "id"},
				},
			},
			nil).Times(1)
		wc.EXPECT().ReadAuthorizationModels(gomock.Any(), gomock.Any()).Return(
			&openfga.ReadAuthorizationModelsResponse{
				AuthorizationModels: []*openfga.AuthorizationModel{
					{Id: "id"},
				},
			},
			nil).Times(1)
		wc.EXPECT().ListObjects(gomock.Any(), gomock.Any(), gomock.Any()).Return(&openfga.ListObjectsResponse{}, nil).Times(1)
		aclClient := acl.NewACLClient(wc, nil, rc)

		s := service.NewService(mockRepository, nil, nil, nil, nil, nil, nil, &aclClient, "")

		_, _, _, err = s.ListModels(context.Background(), 100, "", modelPB.View_VIEW_FULL, nil,
			filtering.Filter{}, false, ordering.OrderBy{})
		assert.NoError(t, err)
	})
}

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
		s := service.NewService(mockRepository, nil, nil, nil, nil, nil, nil, nil, "")

		_, err := s.GetModelDefinition(context.Background(), "github")
		assert.NoError(t, err)
	})

	t.Run("GetModelDefinitionByUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetModelDefinitionByUID(ModelDefinition).
			Return(&datamodel.ModelDefinition{}, nil).
			Times(1)
		s := service.NewService(mockRepository, nil, nil, nil, nil, nil, nil, nil, "")

		_, err := s.GetModelDefinitionByUID(context.Background(), ModelDefinition)
		assert.NoError(t, err)
	})
}

func TestListModelDefinitions(t *testing.T) {
	t.Run("TestListModelDefinition", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			ListModelDefinitions(modelPB.View_VIEW_FULL, int64(100), "").
			Return([]*datamodel.ModelDefinition{}, "", int64(100), nil).
			Times(1)
		s := service.NewService(mockRepository, nil, nil, nil, nil, nil, nil, nil, "")

		_, _, _, err := s.ListModelDefinitions(context.Background(), modelPB.View_VIEW_FULL, int32(100), "")
		assert.NoError(t, err)
	})
}

func TestService_ListNamespaceModelVersions(t *testing.T) {
}
