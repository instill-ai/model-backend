package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/util"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

type ModelParams struct {
	Model datamodel.Model
	Owner string
}

func (w *worker) DeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("DeployModelWorkflow started")

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: 300 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if err := workflow.ExecuteActivity(ctx, w.DeployModelActivity, param).Get(ctx, nil); err != nil {
		return err
	}

	logger.Info("DeployModelWorkflow completed")

	return nil
}

func (w *worker) DeployModelActivity(ctx context.Context, param *ModelParams) error {

	dbModel, err := w.repository.GetModelByUid(param.Owner, param.Model.UID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	modelDef, err := w.repository.GetModelDefinitionByUid(dbModel.ModelDefinitionUid)
	if err != nil {
		return err
	}

	var tritonModels []datamodel.TritonModel
	if tritonModels, err = w.repository.GetTritonModels(dbModel.UID); err != nil {
		return err
	}

	resourceName := util.ConvertModelToResourceName(dbModel.ID)

	// downloading model weight when making inference
	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%s", rdid.String())
	switch modelDef.ID {
	case "github":
		if !config.Config.Server.ItMode && !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var modelConfig datamodel.GitHubModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
				modelSrcDir = util.MODEL_CACHE_DIR + "/" + modelConfig.Repository + modelConfig.Tag
			}

			if err := util.GitHubClone(modelSrcDir, modelConfig, true); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
		}
	case "huggingface":
		if !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var modelConfig datamodel.HuggingFaceModelConfiguration
			if err := json.Unmarshal(dbModel.Configuration, &modelConfig); err != nil {
				return err
			}

			if config.Config.Cache.Model { // cache model into ~/.cache/instill/models
				modelSrcDir = util.MODEL_CACHE_DIR + "/" + modelConfig.RepoId
			}

			if config.Config.Server.ItMode { // use local model to remove internet connection issue while integration testing
				if err = util.HuggingFaceExport(modelSrcDir, datamodel.HuggingFaceModelConfiguration{
					RepoId: "assets/tiny-vit-random",
				}, dbModel.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			} else {
				if err = util.HuggingFaceExport(modelSrcDir, modelConfig, dbModel.ID); err != nil {
					_ = os.RemoveAll(modelSrcDir)
					return err
				}
			}

			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}

			if err := util.UpdateModelConfig(config.Config.TritonServer.ModelStore, tritonModels); err != nil {
				return err
			}
		}
	case "artivc":
		if !config.Config.Server.ItMode && !util.HasModelWeightFile(config.Config.TritonServer.ModelStore, tritonModels) {
			var modelConfig datamodel.ArtiVCModelConfiguration
			err = json.Unmarshal([]byte(dbModel.Configuration), &modelConfig)
			if err != nil {
				return err
			}

			err = util.ArtiVCClone(modelSrcDir, modelConfig, true)
			if err != nil {
				_ = os.RemoveAll(modelSrcDir)
				return err
			}
			if err := util.CopyModelFileToModelRepository(config.Config.TritonServer.ModelStore, modelSrcDir, tritonModels); err != nil {
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
		if _, err = w.triton.LoadModelRequest(tModel.Name); err == nil {
			continue
		}
		if _, e := w.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
			Resource: &controllerPB.Resource{
				Name: resourceName,
				State: &controllerPB.Resource_ModelState{
					ModelState: modelPB.Model_STATE_ERROR,
				},
				Progress: nil,
			},
		}); e != nil {
			return e
		}
	}

	if tEnsembleModel.Name != "" { // load ensemble model.
		if _, err = w.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
			if _, err = w.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
				Resource: &controllerPB.Resource{
					Name: resourceName,
					State: &controllerPB.Resource_ModelState{
						ModelState: modelPB.Model_STATE_ERROR,
					},
					Progress: nil,
				},
			}); err != nil {
				return err
			}
		}
	}

	_, err = w.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ONLINE,
			},
			Progress: nil,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (w *worker) UnDeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("UnDeployModelWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if err := workflow.ExecuteActivity(ctx, w.UnDeployModelActivity, param).Get(ctx, nil); err != nil {
		return err
	}

	logger.Info("UnDeployModelWorkflow completed")

	return nil
}

func (w *worker) UnDeployModelActivity(ctx context.Context, param *ModelParams) error {
	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = w.repository.GetTritonModels(param.Model.UID); err != nil {
		return err
	}

	dbModel, err := w.repository.GetModelByUid(param.Owner, param.Model.UID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	resourceName := util.ConvertModelToResourceName(dbModel.ID)

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = w.triton.UnloadModelRequest(tm.Name); err != nil {
			if _, err := w.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
				Resource: &controllerPB.Resource{
					Name: resourceName,
					State: &controllerPB.Resource_ModelState{
						ModelState: modelPB.Model_STATE_ERROR,
					},
					Progress: nil,
				},
			}); err != nil {
				return err
			}
			return err
		}
	}

	_, err = w.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ModelState{
				ModelState: modelPB.Model_STATE_ONLINE,
			},
			Progress: nil,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (w *worker) CreateModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	fmt.Println("CreateModelWorkflow started")
	logger := workflow.GetLogger(ctx)
	logger.Info("CreateModelWorkflow started")

	if err := w.repository.CreateModel(param.Model); err != nil {
		return err
	}

	logger.Info("CreateModelWorkflow completed")

	return nil
}
