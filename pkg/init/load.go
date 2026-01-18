package init

import (
	"fmt"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

const (
	seedDir = "config/init/%s/seed/%s"
)

func LoadDefinitions(modelDefs *[]*modelpb.ModelDefinition) error {

	modelDefsFiles := []string{
		fmt.Sprintf(seedDir, "instill", "model_definitions.json"),
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
