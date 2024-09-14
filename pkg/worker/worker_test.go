package worker_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/mock"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/worker"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func TestWorker_TriggerModelActivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := minimock.NewController(t)

	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	repo := mock.NewRepositoryMock(mc)

	mockMinio := mock.NewMinioIMock(mc)
	mockMinio.UploadFileBytesMock.Return("", nil, nil)

	t.Run("Task_TASK_TEXT_GENERATION", func(t *testing.T) {
		param := &worker.TriggerModelActivityRequest{}
		param.ModelUID, _ = uuid.NewV4()
		param.TriggerUID, _ = uuid.NewV4()
		param.UserUID, _ = uuid.NewV4()
		param.RequesterUID, _ = uuid.NewV4()
		param.OwnerUID, _ = uuid.NewV4()
		param.ModelID = "ModelID"
		param.OwnerType = string(resource.User)
		param.ModelVersion = datamodel.ModelVersion{
			Version:  "Version",
			ModelUID: param.ModelUID,
		}
		param.Task = commonpb.Task_TASK_CHAT
		param.Visibility = datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PRIVATE)

		uid, _ := uuid.NewV4()
		modelTrigger := &datamodel.ModelRun{
			BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: uid},
			ModelUID:             param.ModelUID,
			ModelVersion:         param.ModelVersion.Version,
			Status:               datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
			RequesterUID:         param.RequesterUID,
			InputReferenceID:     "inputReferenceID",
		}
		param.RunLog = modelTrigger

		mockRay := mock.NewRayMock(mc)
		ctx := context.Background()

		state := modelpb.State_STATE_ACTIVE
		mockRay.ModelReadyMock.Times(1).Return(&state, "", 1, nil)
		mockRay.ModelInferRequestMock.Times(1).
			Return(&rayserver.CallResponse{
				TaskOutputs: []*structpb.Struct{},
			}, nil)
		mockMinio.GetFileMock.Expect(
			minimock.AnyContext,
			modelTrigger.InputReferenceID,
		).Return(
			[]byte("{}"),
			nil,
		)

		repo.UpdateModelRunMock.Times(1).Return(nil)

		w := worker.NewWorker(rc, mockRay, repo, mockMinio, nil)
		err := w.TriggerModelActivity(ctx, param)
		require.NoError(t, err)
	})

	t.Run("when model is offline", func(t *testing.T) {
		param := &worker.TriggerModelActivityRequest{}
		param.UserUID, _ = uuid.NewV4()
		param.OwnerUID, _ = uuid.NewV4()
		param.TriggerUID, _ = uuid.NewV4()
		param.ModelID = "ModelID"
		param.OwnerType = string(resource.User)
		param.ModelVersion = datamodel.ModelVersion{
			Version:  "Version",
			ModelUID: param.ModelUID,
		}
		param.Task = commonpb.Task_TASK_CHAT
		param.Visibility = datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PRIVATE)

		uid, _ := uuid.NewV4()
		modelTrigger := &datamodel.ModelRun{
			BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: uid},
			ModelUID:             param.ModelUID,
			ModelVersion:         param.ModelVersion.Version,
			Status:               datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
			RequesterUID:         param.RequesterUID,
			InputReferenceID:     "inputReferenceID",
		}
		param.RunLog = modelTrigger

		mockRay := mock.NewRayMock(mc)
		ctx := context.Background()

		mockRay.ModelReadyMock.Times(1).Return(modelpb.State_STATE_OFFLINE.Enum(), "", 1, nil)

		w := worker.NewWorker(rc, mockRay, repo, nil, nil)
		err = w.TriggerModelActivity(ctx, param)
		require.ErrorContains(t, err, "model upscale failed")
	})
}
