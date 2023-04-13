package worker

import (
	"context"
	"time"

	"github.com/allegro/bigcache"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
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
	cache            *bigcache.BigCache
	repository       repository.Repository
	triton           triton.Triton
	controllerClient controllerPB.ControllerPrivateServiceClient
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository, t triton.Triton, c controllerPB.ControllerPrivateServiceClient) Worker {
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))

	return &worker{
		cache:            cache,
		repository:       r,
		triton:           t,
		controllerClient: c,
	}
}
