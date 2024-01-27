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
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
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

	dbModel, err := w.repository.GetModelByUIDAdmin(ctx, param.Model.UID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	modelDef, err := w.repository.GetModelDefinitionByUID(dbModel.ModelDefinitionUID)
	if err != nil {
		return err
	}

	var inferenceModels []*datamodel.InferenceModel
	if inferenceModels, err = w.repository.GetInferenceModels(dbModel.UID); err != nil {
		return err
	}

	// downloading model weight when making inference
	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
	switch modelDef.ID {
	case "github":
		if !config.Config.Server.ItMode.Enabled && !utils.HasDVCWeightFile(config.Config.TritonServer.ModelStore, inferenceModels) {
			var modelConfig datamodel.GitHubModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model.Enabled { // cache model into ~/.cache/instill/models
				modelSrcDir = config.Config.Cache.Model.CacheDir + "/" + fmt.Sprintf("%s_%s", modelConfig.Repository, modelConfig.Tag)
			}

			if err := utils.GitHubClone(modelSrcDir, modelConfig, true, w.redisClient); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, inferenceModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	case "huggingface":
		if !utils.HasModelWeightFile(config.Config.TritonServer.ModelStore, inferenceModels) {
			var modelConfig datamodel.HuggingFaceModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model.Enabled { // cache model into ~/.cache/instill/models
				modelSrcDir = config.Config.Cache.Model.CacheDir + "/" + modelConfig.RepoID
			}

			if config.Config.Server.ItMode.Enabled { // use local model to remove internet connection issue while integration testing
				if err = utils.HuggingFaceExport(modelSrcDir, datamodel.HuggingFaceModelConfiguration{
					RepoID: "assets/tiny-vit-random",
				}, dbModel.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			} else {
				if err = utils.HuggingFaceExport(modelSrcDir, modelConfig, dbModel.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			}

			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, inferenceModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}

			if err := utils.UpdateModelConfig(config.Config.TritonServer.ModelStore, inferenceModels); err != nil {
				return err
			}
		}
	case "artivc":
		if !config.Config.Server.ItMode.Enabled && !utils.HasModelWeightFile(config.Config.TritonServer.ModelStore, inferenceModels) {
			var modelConfig datamodel.ArtiVCModelConfiguration
			err = json.Unmarshal([]byte(dbModel.Configuration), &modelConfig)
			if err != nil {
				return err
			}

			err = utils.ArtiVCClone(modelSrcDir, modelConfig, true)
			if err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, inferenceModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	}

	if !config.Config.Cache.Model.Enabled {
		_ = os.RemoveAll(modelSrcDir)
	}

	iEnsembleModel, _ := w.repository.GetInferenceEnsembleModel(param.Model.UID)
	switch iEnsembleModel.Platform {
	case "ensemble":
		for _, tModel := range inferenceModels {
			if iEnsembleModel.Name != "" && iEnsembleModel.Name == tModel.Name { // load ensemble model last.
				continue
			}
			if _, err = w.triton.LoadModelRequest(ctx, tModel.Name); err == nil {
				continue
			}
			logger.Error(fmt.Sprintf("triton model deployment failed: %v", err))
			return err
		}
		if iEnsembleModel.Name != "" { // load ensemble model.
			if _, err = w.triton.LoadModelRequest(ctx, iEnsembleModel.Name); err != nil {
				logger.Error(fmt.Sprintf("triton model deployment failed: %v", err))
				return err
			}
		}
	case "ray":
		name := filepath.Join(iEnsembleModel.Name, fmt.Sprint(iEnsembleModel.Version))
		if err = w.ray.DeployModel(name); err != nil {
			logger.Error(fmt.Sprintf("ray model deployment failed: %v", err))
			return err
		}
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

	inferenceModels, _ := w.repository.GetInferenceModels(param.Model.UID)
	iEnsembleModel, _ := w.repository.GetInferenceEnsembleModel(param.Model.UID)
	switch iEnsembleModel.Platform {
	case "ensemble":
		for _, rModel := range inferenceModels {
			if _, err := w.triton.UnloadModelRequest(ctx, rModel.Name); err != nil {
				logger.Error(fmt.Sprintf("triton model undeployment failed: %v", err))
			}
		}
	case "ray":
		name := filepath.Join(iEnsembleModel.Name, fmt.Sprint(iEnsembleModel.Version))
		if err := w.ray.UndeployModel(name); err != nil {
			logger.Error(fmt.Sprintf("ray model undeployment failed: %v", err))
		}
		// TODO: fix no return here by properly check in ray app is serving
		// if not return nil even if workflow failed
	}

	logger.Info("UnDeployModelActivity completed")

	return nil
}

func (w *worker) CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CreateModelWorkflow started")

	if err := w.repository.CreateUserModel(param.Model); err != nil {
		if e, ok := status.FromError(err); ok {
			if e.Code() != codes.AlreadyExists {
				return err
			} else {
				logger.Info("Model already existed, CreateModelWorkflow completed")
				return nil
			}
		}
	}

	logger.Info("CreateModelWorkflow completed")

	return nil
}
