package init

import (
	"encoding/json"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var EnumRegistry = map[string]map[string]int32{
	"release_stage": modelPB.ReleaseStage_value,
}

// unmarshalModelPB unmarshals a slice of JSON object into a Protobuf Message Go struct element by element
// See: https://github.com/golang/protobuf/issues/675#issuecomment-411182202
func unmarshalModelPB(jsonSliceMap any, pb any) error {

	pj := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	if v, ok := jsonSliceMap.([]map[string]any); ok {
		for _, vv := range v {

			b, err := json.Marshal(vv)
			if err != nil {
				return err
			}

			if pb, ok := pb.(*[]*modelPB.ModelDefinition); ok {
				modelDef := modelPB.ModelDefinition{}
				if err := pj.Unmarshal(b, &modelDef); err != nil {
					return err
				}
				*pb = append(*pb, &modelDef)
			}
		}
	}

	return nil
}

func processJSONSliceMap(filename string) ([]map[string]any, error) {

	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var jsonSliceMap []map[string]any
	if err := json.Unmarshal(file, &jsonSliceMap); err != nil {
		return nil, err
	}

	convertAllJSONKeySnakeCase(jsonSliceMap)
	convertAllJSONEnumValueToProtoStyle(EnumRegistry, jsonSliceMap)

	return jsonSliceMap, nil
}
