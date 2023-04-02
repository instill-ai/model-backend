package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/util"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

type ModelParams struct {
	Model datamodel.Model
	Owner string
}

func (w *worker) AddSearchAttributeWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("AddSearchAttributeWorkflow started")

	// Upsert search attributes.
	modelUID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	modelInstanceUID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	attributes := map[string]interface{}{
		"Type":             util.OperationTypeHealthCheck,
		"ModelUID":         modelUID.String(),
		"ModelInstanceUID": modelInstanceUID.String(),
		"Owner":            "",
	}

	err = workflow.UpsertSearchAttributes(ctx, attributes)
	if err != nil {
		return err
	}

	logger.Info("AddSearchAttributeWorkflow completed")

	return nil
}

func (w *worker) DeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("DeployModelWorkflow started")

	// Upsert search attributes.
	attributes := map[string]interface{}{
		"Type":     util.OperationTypeDeploy,
		"ModelUID": param.Model.UID.String(),
		"Owner":    strings.TrimPrefix(param.Owner, "users/"),
	}

	err := workflow.UpsertSearchAttributes(ctx, attributes)
	if err != nil {
		return err
	}

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
		if e := w.repository.UpdateModel(param.Model.UID, datamodel.Model{
			State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
		}); e != nil {
			return e
		}
	}

	if tEnsembleModel.Name != "" { // load ensemble model.
		if _, err = w.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
			if e := w.repository.UpdateModel(param.Model.UID, datamodel.Model{
				State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
			}); e != nil {
				return e
			}
		}
	}

	if err = w.repository.UpdateModel(param.Model.UID, datamodel.Model{
		State: datamodel.ModelState(modelPB.Model_STATE_ONLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (w *worker) UnDeployModelWorkflow(ctx workflow.Context, param *ModelParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("UnDeployModelWorkflow started")

	// Upsert search attributes.
	attributes := map[string]interface{}{
		"Type":     util.OperationTypeUnDeploy,
		"ModelUID": param.Model.UID.String(),
		"Owner":    strings.TrimPrefix(param.Owner, "users/"),
	}

	err := workflow.UpsertSearchAttributes(ctx, attributes)
	if err != nil {
		return err
	}

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

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = w.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err1 := w.repository.UpdateModel(param.Model.UID, datamodel.Model{
				State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
			}); err1 != nil {
				return err1
			}
			return err
		}
	}

	if err := w.repository.UpdateModel(param.Model.UID, datamodel.Model{
		State: datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
	}); err != nil {
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

	dbModel, err := w.repository.GetModelById(param.Owner, param.Model.ID, modelPB.View_VIEW_BASIC)
	if err != nil {
		return err
	}

	// Upsert search attributes.
	attributes := map[string]interface{}{
		"Type":     util.OperationTypeCreate,
		"ModelUID": dbModel.UID,
		"Owner":    strings.TrimPrefix(param.Owner, "users/"),
	}

	err = workflow.UpsertSearchAttributes(ctx, attributes)
	if err != nil {
		return err
	}

	logger.Info("CreateModelWorkflow completed")

	return nil
}
