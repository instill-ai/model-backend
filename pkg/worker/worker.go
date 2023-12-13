package worker

import (
	"context"

	"github.com/go-redis/redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"

	controllerPB "github.com/instill-ai/protogen-go/model/controller/v1alpha"
)

// Namespace is the Temporal namespace for model-backend
const Namespace = "model-backend"

// TaskQueue is the Temporal task queue name for model-backend
const TaskQueue = "model-backend"

// Worker interface
type Worker interface {
	DeployModelWorkflow(ctx workflow.Context, param *ModelParams) error
	DeployModelActivity(ctx context.Context, param *ModelParams) error
	UnDeployModelWorkflow(ctx workflow.Context, param *ModelParams) error
	UnDeployModelActivity(ctx context.Context, param *ModelParams) error
	CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	redisClient      *redis.Client
	repository       repository.Repository
	ray              ray.Ray
	triton           triton.Triton
	controllerClient controllerPB.ControllerPrivateServiceClient
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository, rc *redis.Client, t triton.Triton, c controllerPB.ControllerPrivateServiceClient, ra ray.Ray) Worker {

	return &worker{
		repository:       r,
		redisClient:      rc,
		ray:              ra,
		triton:           t,
		controllerClient: c,
	}
}
