package worker

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

type ModelParams struct {
	UserID string
	Model  *datamodel.Model
}

var tracer = otel.Tracer("model-backend.temporal.tracer")

func (w *worker) DeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("DeployModelWorkflow started")

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: 300 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if err := workflow.ExecuteActivity(ctx, w.DeployModelActivity, param).Get(ctx, nil); err != nil {
		return err
	}

	logger.Info("DeployModelWorkflow completed")

	return nil
}

func (w *worker) DeployModelActivity(ctx context.Context, param *ModelParams) error {

	ctx, span := tracer.Start(ctx, "DeployModelActivity",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := activity.GetLogger(ctx)

	logger.Info("DeployModelActivity started")

	// TODO: deprecated

	logger.Info("DeployModelActivity completed")

	return nil
}

func (w *worker) UnDeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("UnDeployModelWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if err := workflow.ExecuteActivity(ctx, w.UnDeployModelActivity, param).Get(ctx, nil); err != nil {
		return err
	}

	logger.Info("UnDeployModelWorkflow completed")

	return nil
}

func (w *worker) UnDeployModelActivity(ctx context.Context, param *ModelParams) error {

	ctx, span := tracer.Start(ctx, "UnDeployModelActivity",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("UnDeployModelActivity started")

	// TODO: deprecated

	logger.Info("UnDeployModelActivity completed")

	return nil
}

func (w *worker) CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CreateModelWorkflow started")

	// TODO: deprecated

	logger.Info("CreateModelWorkflow completed")

	return nil
}
