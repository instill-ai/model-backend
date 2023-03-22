package init

import (
	"fmt"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

const (
	seedDir = "config/init/%s/seed/%s"
)

func LoadDefinitions(modelDefs *[]*modelPB.ModelDefinition) error {

	modelDefsFiles := []string{
		fmt.Sprintf(seedDir, "instill", "model_definitions.yaml"),
	}

	for _, filename := range modelDefsFiles {
		if jsonSliceMap, err := processJSONSliceMap(filename); err == nil {
			if err := unmarshalModelPB(jsonSliceMap, modelDefs); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
