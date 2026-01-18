package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/x/minio"
)

// TaskQueue is the Temporal task queue name for model-backend
const TaskQueue = "model-backend"

// Worker interface
type Worker interface {
	TriggerModelWorkflow(ctx workflow.Context, param *TriggerModelWorkflowRequest) error
	TriggerModelActivity(ctx context.Context, param *TriggerModelActivityRequest) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	redisClient         *redis.Client
	ray                 ray.Ray
	minioClient         minio.Client
	repository          repository.Repository
	influxDBWriteClient api.WriteAPI
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	rc *redis.Client,
	ra ray.Ray,
	repo repository.Repository,
	i api.WriteAPI,
	minioClient minio.Client,
) Worker {
	return &worker{
		redisClient:         rc,
		ray:                 ra,
		minioClient:         minioClient,
		repository:          repo,
		influxDBWriteClient: i,
	}
}
