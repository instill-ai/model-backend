package worker_test

//go:generate mockgen -destination mock_ray_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/ray Ray

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"

	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/mock"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/worker"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	taskv1alpha "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func TestWorker_TriggerModelWorkflow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	t.Run("Task_TASK_TEXT_GENERATION", func(t *testing.T) {
		// todo: fix input workflow.Context for unit test
		t.SkipNow()

		param := &worker.TriggerModelWorkflowRequest{}
		mockRay := NewMockRay(ctrl)

		w := worker.NewWorker(rc, mockRay, nil, nil, nil)
		err = w.TriggerModelWorkflow(workflow.Context(nil), param)
		require.NoError(t, err)
	})
}

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

	repo := mock.NewMockRepository(ctrl)

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
		param.InputKey = "InputKey"
		param.Task = taskv1alpha.Task_TASK_CHAT
		param.Visibility = datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE)
		param.Source = datamodel.TriggerSource(runpb.RunSource_RUN_SOURCE_API)

		mockRay := NewMockRay(ctrl)
		ctx := context.Background()

		state := modelPB.State_STATE_ACTIVE
		mockRay.EXPECT().
			ModelReady(
				gomock.Any(),
				fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID),
				param.ModelVersion.Version,
			).Return(
			&state,
			"",
			1,
			nil,
		).Times(1)
		mockRay.EXPECT().ModelInferRequest(
			gomock.Any(),
			param.Task,
			gomock.Any(),
			fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID),
			param.ModelVersion.Version,
		).Return(&rayserver.CallResponse{
			TaskOutputs: []*structpb.Struct{},
		}, nil).Times(1)

		rc.Set(ctx, param.InputKey, "{}", 30*time.Second)

		uid, _ := uuid.NewV4()
		modelTrigger := &datamodel.ModelTrigger{
			BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: uid},
			ModelUID:             param.ModelUID,
			ModelVersion:         param.ModelVersion.Version,
			Status:               datamodel.TriggerStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
			Source:               param.Source,
			RequesterUID:         param.RequesterUID,
			InputReferenceID:     param.InputReferenceID,
		}
		param.RunLog = modelTrigger
		repo.EXPECT().UpdateModelTrigger(gomock.Any(), gomock.Any()).Return(nil).Times(1)

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
		param.InputKey = "InputKey"
		param.Task = taskv1alpha.Task_TASK_CHAT
		param.Visibility = datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE)
		param.Source = datamodel.TriggerSource(runpb.RunSource_RUN_SOURCE_API)

		mockRay := NewMockRay(ctrl)
		ctx := context.Background()

		mockRay.EXPECT().
			ModelReady(
				gomock.Any(),
				fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID),
				param.ModelVersion.Version,
			).Return(
			modelPB.State_STATE_OFFLINE.Enum(),
			"",
			1,
			nil,
		).Times(1)

		uid, _ := uuid.NewV4()
		modelTrigger := &datamodel.ModelTrigger{
			BaseStaticHardDelete: datamodel.BaseStaticHardDelete{UID: uid},
			ModelUID:             param.ModelUID,
			ModelVersion:         param.ModelVersion.Version,
			Status:               datamodel.TriggerStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
			Source:               param.Source,
			RequesterUID:         param.RequesterUID,
			InputReferenceID:     param.InputReferenceID,
		}
		param.RunLog = modelTrigger
		// repo.EXPECT().UpdateModelTrigger(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		w := worker.NewWorker(rc, mockRay, repo, nil, nil)
		err = w.TriggerModelActivity(ctx, param)
		require.ErrorContains(t, err, "model upscale failed")
	})
}
