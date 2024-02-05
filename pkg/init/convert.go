package init

import (
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
)

// convertAllJSONKeySnakeCase traverses a JSON object to replace all keys to snake_case except for the JSON Schema object.
func convertAllJSONKeySnakeCase(i any) {
	switch v := i.(type) {
	case map[string]any:
		for k, vv := range v {
			if k == "minLength" || k == "maxLength" || k == "minProperties" || k == "maxProperties" { // TODO: json schema validator failed with snake_case, need further check
				continue
			}
			sc := strcase.ToSnake(k)
			if sc != k {
				v[sc] = v[k]
				delete(v, k)
			}

			convertAllJSONKeySnakeCase(vv)
		}
	case []map[string]any:
		for _, vv := range v {
			convertAllJSONKeySnakeCase(vv)
		}
	}
}

// convertAllJSONEnumValueToProtoStyle converts lowercase enum value to the Protobuf naming convention where the enum type is always prefixed and is UPPERCASE snake_case.
// For examples:
// - api in a Protobuf `Enum SourceType` type will be converted to SOURCE_TYPE_API
// - oauth2.0  in a Protobuf `Enum AuthFlowType` type will be converted to AUTH_FLOW_TYPE_OAUTH2_0
func convertAllJSONEnumValueToProtoStyle(enumRegistry map[string]map[string]int32, i any) {
	switch v := i.(type) {
	case map[string]any:
		for k, vv := range v {
			if _, ok := enumRegistry[k]; ok {
				for enumKey := range enumRegistry[k] {
					if reflect.TypeOf(vv).Kind() == reflect.Slice { // repeated enum type
						for kk, vvv := range vv.([]any) {
							if strings.ReplaceAll(vvv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
								vv.([]any)[kk] = enumKey
							}
						}
					} else if strings.ReplaceAll(vv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
						v[k] = enumKey
					}
				}
			}
			convertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	case []map[string]any:
		for _, vv := range v {
			convertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	}
}
