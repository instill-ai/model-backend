package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"
)

type ModelParams struct {
	Model *datamodel.Model
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

	modelDef, err := w.repository.GetModelDefinitionByUID(param.Model.ModelDefinitionUID)
	if err != nil {
		return err
	}

	// downloading model weight when making inference
	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
	switch modelDef.ID {
	case "github":
		if !config.Config.Server.ItMode.Enabled && !utils.HasDVCWeightFile(config.Config.RayServer.ModelStore, param.Model) {
			var modelConfig datamodel.GitHubModelConfiguration
			if err := json.Unmarshal(param.Model.Configuration, &modelConfig); err != nil {
				return err
			}

			redisRepoKey := fmt.Sprintf("model_cache:%s:%s", modelConfig.Repository, modelConfig.Tag)
			if config.Config.Cache.Model.Enabled { // cache model into ~/.cache/instill/models
				modelSrcDir = config.Config.Cache.Model.CacheDir + "/" + fmt.Sprintf("%s_%s", modelConfig.Repository, modelConfig.Tag)
			}

			if err := utils.GitHubClone(modelSrcDir, modelConfig, true, w.redisClient, redisRepoKey); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				w.redisClient.Del(ctx, redisRepoKey)
				return err
			}
			if err := utils.CopyModelFileToModelRepository(config.Config.RayServer.ModelStore, modelSrcDir, param.Model); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				w.redisClient.Del(ctx, redisRepoKey)
				return err
			}
		}
	case "container":
		name := filepath.Join(param.Model.Owner, param.Model.ID)
		if err = w.ray.UpdateContainerizedModel(name, param.Model.ID, true); err != nil {
			logger.Error(fmt.Sprintf("containerized ray model deployment failed: %v", err))
			return err
		}

		logger.Info("DeployModelActivity completed")
		return nil
	case "huggingface":
		if !utils.HasModelWeightFile(config.Config.RayServer.ModelStore, param.Model) {
			var modelConfig datamodel.HuggingFaceModelConfiguration
			if err := json.Unmarshal(param.Model.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model.Enabled { // cache model into ~/.cache/instill/models
				modelSrcDir = config.Config.Cache.Model.CacheDir + "/" + modelConfig.RepoID
			}

			if config.Config.Server.ItMode.Enabled { // use local model to remove internet connection issue while integration testing
				if err = utils.HuggingFaceExport(modelSrcDir, datamodel.HuggingFaceModelConfiguration{
					RepoID: "assets/tiny-vit-random",
				}, param.Model.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			} else {
				if err = utils.HuggingFaceExport(modelSrcDir, modelConfig, param.Model.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			}

			if err := utils.CopyModelFileToModelRepository(config.Config.RayServer.ModelStore, modelSrcDir, param.Model); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	case "artivc":
		if !config.Config.Server.ItMode.Enabled && !utils.HasModelWeightFile(config.Config.RayServer.ModelStore, param.Model) {
			var modelConfig datamodel.ArtiVCModelConfiguration
			err = json.Unmarshal([]byte(param.Model.Configuration), &modelConfig)
			if err != nil {
				return err
			}

			err = utils.ArtiVCClone(modelSrcDir, modelConfig, true)
			if err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := utils.CopyModelFileToModelRepository(config.Config.RayServer.ModelStore, modelSrcDir, param.Model); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	}

	if !config.Config.Cache.Model.Enabled {
		_ = os.RemoveAll(modelSrcDir)
	}

	name := filepath.Join(param.Model.Owner, param.Model.ID)
	if err = w.ray.DeployModel(name); err != nil {
		logger.Error(fmt.Sprintf("ray model deployment failed: %v", err))
		return err
	}

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

	modelDef, err := w.repository.GetModelDefinitionByUID(param.Model.ModelDefinitionUID)
	if err != nil {
		return err
	}

	if modelDef.ID == "container" {
		name := filepath.Join(param.Model.Owner, param.Model.ID)
		if err := w.ray.UpdateContainerizedModel(name, param.Model.ID, false); err != nil {
			logger.Error(fmt.Sprintf("containerized ray model undeployment failed: %v", err))
		}
		logger.Info("UnDeployModelActivity completed")
		return nil
	}

	name := filepath.Join(param.Model.Owner, param.Model.ID)
	if err := w.ray.UndeployModel(name); err != nil {
		logger.Error(fmt.Sprintf("ray model undeployment failed: %v", err))
	}
	// TODO: fix no return here by properly check in ray app is serving
	// if not return nil even if workflow failed

	logger.Info("UnDeployModelActivity completed")

	return nil
}

func (w *worker) CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CreateModelWorkflow started")

	c, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.repository.CreateNamespaceModel(c, param.Model.Owner, param.Model); err != nil {
		if e, ok := status.FromError(err); ok {
			if e.Code() != codes.AlreadyExists {
				return err
			} else {
				logger.Info("Model already existed, CreateModelWorkflow completed")
				return nil
			}
		}
	}

	dbCreatedModel, err := w.repository.GetNamespaceModelByID(c, param.Model.Owner, param.Model.ID, false)
	if err != nil {
		return err
	}

	nsType, ownerUID, err := resource.GetNamespaceTypeAndUID(param.Model.Owner)
	if err != nil {
		return err
	}
	ownerType := nsType[0 : len(nsType)-1]

	err = w.aclClient.SetOwner("model_", dbCreatedModel.UID, ownerType, ownerUID)
	if err != nil {
		return err
	}

	logger.Info("CreateModelWorkflow completed")

	return nil
}
