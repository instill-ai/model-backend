package worker

import (
	"context"
	"time"

	"github.com/allegro/bigcache"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/repository"
)

// Namespace is the Temporal namespace for model-backend
const Namespace = "model-backend"

// TaskQueue is the Temporal task queue name for model-backend
const TaskQueue = "model-backend"

// Worker interface
type Worker interface {
	DeployModelWorkflow(ctx workflow.Context, param *ModelInstanceParams) error
	DeployModelActivity(ctx context.Context, param *ModelInstanceParams) error
	UnDeployModelWorkflow(ctx workflow.Context, param *ModelInstanceParams) error
	UnDeployModelActivity(ctx context.Context, param *ModelInstanceParams) error
	CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error
	AddSearchAttributeWorkflow(ctx workflow.Context) error
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
