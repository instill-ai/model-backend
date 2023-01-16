package worker

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/allegro/bigcache"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/repository"
)

// TaskQueue is the task queue name for connector-backend
const TaskQueue = "model-backend"
const Namespace = "model-backend"

// Worker interface
type Worker interface {
	DeployModelWorkflow(ctx workflow.Context, param *ModelInstanceParams) error
	DeployModelActivity(ctx context.Context, param *ModelInstanceParams) error
	UnDeployModelWorkflow(ctx workflow.Context, param *ModelInstanceParams) error
	UnDeployModelActivity(ctx context.Context, param *ModelInstanceParams) error
	CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error
	SearchAttributeReadyWorkflow(ctx workflow.Context) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	cache      *bigcache.BigCache
	repository repository.Repository
	triton     triton.Triton
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository, t triton.Triton) Worker {
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))

	return &worker{
		cache:      cache,
		repository: r,
		triton:     t,
	}
}
