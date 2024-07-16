package worker_test

//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/repository Repository
//go:generate mockgen -destination mock_ray_test.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/ray Ray

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	taskv1alpha "github.com/instill-ai/protogen-go/common/task/v1alpha"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/worker"
)

func TestWorker_TriggerModelActivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	t.Run("Task_TASK_TEXT_GENERATION", func(t *testing.T) {
		param := &worker.TriggerModelActivityRequest{}
		param.UserUID, _ = uuid.NewV4()
		param.OwnerUID, _ = uuid.NewV4()
		param.ModelID = "ModelID"
		param.OwnerType = string(resource.User)
		param.ModelVersion = datamodel.ModelVersion{
			Version:  "Version",
			ModelUID: param.ModelUID,
		}
		param.ParsedInputKey = "ParsedInputKey"
		param.Task = taskv1alpha.Task_TASK_TEXT_GENERATION

		mockRay := NewMockRay(ctrl)
		ctx := context.Background()
		name := "text"

		var contents [][]byte
		err = json.Unmarshal([]byte(`["RAEAADx8c3lzdGVtfD4KWW91IGFyZSBhIGZyaWVuZGx5IGNoYXRib3Q8L3M+Cjx8dXNlcnw+CndoYXQgZG9lcyB0aGUgY29tcGFueSB0ZXNsYSBkbz88L3M+Cjx8YXNzaXN0YW50fD4KVGhlIGNvbXBhbnkgVGVzbGEgZG9lcyBub3QgaGF2ZSBhIHBoeXNpY2FsIHByZXNlbmNlLiBIb3dldmVyLCBpdCBpcyBhIHRlY2hub2xvZ3kgY29tcGFueSB0aGF0IGRldmVsb3BzIGFuZCBtYW51ZmFjdHVyZXMgZWxlY3RyaWMgdmVoaWNsZXMgKEVWcyksIGVuZXJneSBzdG9yYWdlIHN5c3RlbXMsIGFuZCBzb2xhciBwYW5lbHMuIFRlc2xhJ3MgcHJpbWFyeSBmb2N1cyBpcyBvbiBlbGVjdHJpYw=="]`), &contents)
		require.NoError(t, err)

		mockRay.EXPECT().
			ModelMetadataRequest(
				gomock.Any(),
				fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID),
				param.ModelVersion.Version,
			).
			Return(&rayserver.ModelMetadataResponse{
				Outputs: []*rayserver.ModelMetadataResponse_TensorMetadata{
					{Name: name, Shape: []int64{-1, -1}},
				},
			}).Times(1)
		mockRay.EXPECT().ModelInferRequest(
			gomock.Any(),
			param.Task,
			gomock.Any(),
			fmt.Sprintf("%s/%s/%s", param.OwnerType, param.OwnerUID.String(), param.ModelID),
			param.ModelVersion.Version,
			gomock.Any(),
		).Return(&rayserver.RayServiceCallResponse{
			RawOutputContents: contents,
			Outputs: []*rayserver.InferTensor{
				{Name: name, Shape: []int64{1, 1}},
			},
		}, nil).Times(1)

		rc.Set(ctx, param.ParsedInputKey, "{}", 30*time.Second)

		w := worker.NewWorker(nil, rc, mockRay, nil)
		resp, err := w.TriggerModelActivity(ctx, param)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Contains(t, resp.OutputKey, "async_model_response:")
		require.Contains(t, string(resp.TaskOutputBytes), "You are a friendly chatbot")
	})
}
