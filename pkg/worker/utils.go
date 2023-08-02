package worker

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type PreModelConfig struct {
	ID              string            `json:"id"`
	Description     string            `json:"description"`
	Task            string            `json:"task"`
	ModelDefinition string            `json:"model_definition"`
	Configuration   map[string]string `json:"configuration"`
}

var preDeployModelMap = map[string]map[string]string{
	"mask-rcnn": {
		"uid":                  "37e01efd-acb2-4a69-a67f-bc06599f2473",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_INSTANCE_SEGMENTATION",
	},
	"stomata-mask-rcnn": {
		"uid":                  "2aaf321d-a533-47ea-8616-39a5205797ca",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_INSTANCE_SEGMENTATION",
	},
	"lraspp": {
		"uid":                  "86a903d7-a67b-4318-89a0-b701ed58daed",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_INSTANCE_SEGMENTATION",
	},
	"stable-diffusion-1-5-fp16-txt2img": {
		"uid":                  "40bf6822-8a8c-4f9f-a0b9-c233b5404973",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_TEXT_TO_IMAGE",
	},
	"gpt-2": {
		"uid":                  "6332a144-8e14-4c4d-8d1c-7653bc17d2ff",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_TEXT_GENERATION",
	},
	"mobilenetv2": {
		"uid":                  "0b9c26f2-9d51-43ed-ae0f-30b3f280e7be",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_CLASSIFICATION",
	},
	"yolov7": {
		"uid":                  "82a1a29c-0c3b-4b4e-83a8-7ea6a80bf7b7",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_DETECTION",
	},
	"yolov7-pose": {
		"uid":                  "7ee5e127-3dc1-4c8f-947c-966b6281e772",
		"owner":                "users/eeaf4fe1-5725-4802-98b9-6165bd2b1e58",
		"model_definition_uid": "909c3278-f7d1-461c-9352-87741bef11d3",
		"task":                 "TASK_KEYPOINT",
	},
}

func GetPreDeployGitHubModelUUID(model datamodel.Model) (*datamodel.PreDeployModel, error) {
	var preDeployModelConfigs []PreModelConfig
	err := utils.GetJSON(config.Config.InitModel.Path, &preDeployModelConfigs)
	if err != nil {
		return nil, err
	}

	modelConfig := datamodel.GitHubModelConfiguration{}
	err = json.Unmarshal(model.Configuration, &modelConfig)
	if err != nil {
		return nil, err
	}

	var githubModel *datamodel.PreDeployModel

	if _, found := preDeployModelMap[model.ID]; !found {
		return githubModel, nil
	}

	for _, preDeployModelConfigs := range preDeployModelConfigs {
		if modelConfig.Repository == preDeployModelConfigs.Configuration["repository"] &&
			modelConfig.Tag == preDeployModelConfigs.Configuration["tag"] {

			uid, _ := uuid.FromString(preDeployModelMap[preDeployModelConfigs.ID]["uid"])
			modelDefinitionUID, _ := uuid.FromString(preDeployModelMap[preDeployModelConfigs.ID]["model_definition_uid"])

			githubModel = &datamodel.PreDeployModel{
				BaseStatic: datamodel.BaseStatic{
					UID:        uid,
					CreateTime: model.CreateTime,
					UpdateTime: model.UpdateTime,
					DeleteTime: model.DeleteTime,
				},
				ID:                 model.ID,
				ModelDefinitionUid: modelDefinitionUID,
				Owner:              model.Owner,
				Visibility:         datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC),
				State:              model.State,
				Task:               model.Task,
				Description:        model.Description,
				Configuration:      model.Configuration,
				TritonModels:       model.TritonModels,
			}
		}
	}
	return githubModel, nil
}
