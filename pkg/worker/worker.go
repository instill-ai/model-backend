package worker

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/pkg/minio"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/usage"
)

// TaskQueue is the Temporal task queue name for model-backend
const TaskQueue = "model-backend"

// Worker interface
type Worker interface {
	TriggerModelWorkflow(ctx workflow.Context, param *TriggerModelWorkflowRequest) error
	TriggerModelActivity(ctx context.Context, param *TriggerModelActivityRequest) error
	UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityRequest) (*UploadToMinioActivityResponse, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	redisClient       *redis.Client
	ray               ray.Ray
	minioClient       minio.MinioI
	repository        repository.Repository
	modelUsageHandler usage.ModelUsageHandler
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(rc *redis.Client, ra ray.Ray, repo repository.Repository, minioClient minio.MinioI, modelUsageHandler usage.ModelUsageHandler) Worker {
	if modelUsageHandler == nil {
		modelUsageHandler = usage.NewNoopModelUsageHandler()
	}
	return &worker{
		redisClient:       rc,
		ray:               ra,
		minioClient:       minioClient,
		repository:        repo,
		modelUsageHandler: modelUsageHandler,
	}
}
