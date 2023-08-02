package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"

	controllerPB "github.com/instill-ai/protogen-go/model/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ModelParams struct {
	Model datamodel.Model
	Owner string
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

	dbModel, err := w.repository.GetModelByUID(param.Owner, param.Model.UID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	modelDef, err := w.repository.GetModelDefinitionByUID(dbModel.ModelDefinitionUid)
	if err != nil {
		return err
	}

	var tritonModels []datamodel.TritonModel
	if tritonModels, err = w.repository.GetTritonModels(dbModel.UID); err != nil {
		return err
	}

	resourcePermalink := utils.ConvertModelToResourcePermalink(dbModel.UID.String())

	updateResourceReq := controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State:             &controllerPB.Resource_ModelState{},
			Progress:          nil,
		},
	}

	// downloading model weight when making inference
	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
	switch modelDef.ID {
	case "github":
		if !config.Config.Server.ItMode.Enabled && !utils.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var modelConfig datamodel.GitHubModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
				modelSrcDir = utils.MODEL_CACHE_DIR + "/" + modelConfig.Repository + modelConfig.Tag
			}

			if err := utils.GitHubClone(modelSrcDir, modelConfig, true); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	case "huggingface":
		if !utils.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var modelConfig datamodel.HuggingFaceModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
				modelSrcDir = utils.MODEL_CACHE_DIR + "/" + modelConfig.RepoId
			}

			if config.Config.Server.ItMode.Enabled { // use local model to remove internet connection issue while integration testing
				if err = utils.HuggingFaceExport(modelSrcDir, datamodel.HuggingFaceModelConfiguration{
					RepoId: "assets/tiny-vit-random",
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

			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}

			if err := utils.UpdateModelConfig(config.Config.TritonServer.ModelStore, tritonModels); err != nil {
				return err
			}
		}
	case "artivc":
		if !config.Config.Server.ItMode.Enabled && !utils.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
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
			if err := utils.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	}

	if !config.Config.Cache.Model {
		_ = os.RemoveAll(modelSrcDir)
	}

	tEnsembleModel, _ := w.repository.GetTritonEnsembleModel(param.Model.UID)
	for _, tModel := range tritonModels {
		if tEnsembleModel.Name != "" && tEnsembleModel.Name == tModel.Name { // load ensemble model last.
			continue
		}
		if _, err = w.triton.LoadModelRequest(ctx, tModel.Name); err == nil {
			continue
		}
		updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
			ModelState: modelPB.Model_STATE_ERROR,
		}
		if _, err := w.controllerClient.UpdateResource(ctx, &updateResourceReq); err != nil {
			return err
		}
		return err
	}

	if tEnsembleModel.Name != "" { // load ensemble model.
		if _, err = w.triton.LoadModelRequest(ctx, tEnsembleModel.Name); err != nil {
			updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ERROR,
			}
			if _, err := w.controllerClient.UpdateResource(ctx, &updateResourceReq); err != nil {
				return err
			}
			return err
		}
	}

	updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
		ModelState: modelPB.Model_STATE_ONLINE,
	}
	if _, err := w.controllerClient.UpdateResource(ctx, &updateResourceReq); err != nil {
		return err
	}

	logger.Info("DeployModelActivity completed")

	return nil
}

func (w *worker) UnDeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("UnDeployModelWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
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

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = w.repository.GetTritonModels(param.Model.UID); err != nil {
		return err
	}

	dbModel, err := w.repository.GetModelByUID(param.Owner, param.Model.UID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	resourcePermalink := utils.ConvertModelToResourcePermalink(dbModel.UID.String())

	updateResourceReq := controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State:             &controllerPB.Resource_ModelState{},
			Progress:          nil,
		},
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = w.triton.UnloadModelRequest(ctx, tm.Name); err != nil {
			updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ERROR,
			}
			if _, err := w.controllerClient.UpdateResource(ctx, &updateResourceReq); err != nil {
				return err
			}
			return err
		}
	}

	updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
		ModelState: modelPB.Model_STATE_OFFLINE,
	}
	if _, err := w.controllerClient.UpdateResource(ctx, &updateResourceReq); err != nil {
		return err
	}

	logger.Info("UnDeployModelActivity completed")

	return nil
}

func (w *worker) CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CreateModelWorkflow started")

	updateResourceReq := controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: "",
			State:             &controllerPB.Resource_ModelState{},
			Progress:          nil,
		},
	}

	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	preDeployModel, err := GetPreDeployGitHubModelUUID(param.Model)
	if err != nil {
		return err
	}

	if preDeployModel != nil && !config.Config.Server.ItMode.Enabled {
		if err := w.repository.CreatePreDeployModel(*preDeployModel); err != nil {
			updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ERROR,
			}
		}
	} else {
		if err := w.repository.CreateModel(param.Model); err != nil {
			updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ERROR,
			}
		}
	}

	dbModel, err := w.repository.GetModelByID(param.Owner, param.Model.ID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return err
	}

	resourcePermalink := utils.ConvertModelToResourcePermalink(dbModel.UID.String())

	updateResourceReq.Resource.ResourcePermalink = resourcePermalink
	updateResourceReq.Resource.State = &controllerPB.Resource_ModelState{
		ModelState: modelPB.Model_STATE_OFFLINE,
	}
	if _, err := w.controllerClient.UpdateResource(controllerCtx, &updateResourceReq); err != nil {
		return err
	}

	logger.Info("CreateModelWorkflow completed")

	return nil
}
