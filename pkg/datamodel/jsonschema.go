package datamodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
)

// ModelDefJSONSchema represents the ModelDefinition JSON Schema for validating the payload
var ModelDefJSONSchema *jsonschema.Schema

// ModelJSONSchema represents the Model JSON Schema for validating the payload
var ModelJSONSchema *jsonschema.Schema

var RegionHardwareJSON RegionHardware

var TaskInputJSON map[string]any
var TaskOutputJSON map[string]any

type RegionHardware struct {
	Properties struct {
		Region struct {
			OneOf []struct {
				Const string `json:"const"`
				Title string `json:"title"`
			}
		}
	}
	AllOf []struct {
		If struct {
			Properties struct {
				Region struct {
					Const string `json:"const"`
				}
			}
		}
		Then struct {
			Properties struct {
				Hardware struct {
					OneOf []struct {
						Const string `json:"const"`
						Title string `json:"title"`
					}
					AnyOf []struct {
						Const string `json:"const"`
						Title string `json:"title"`
					}
				}
			}
		}
	}
}

// InitJSONSchema initializes JSON Schema instances with the given files
func InitJSONSchema(ctx context.Context) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	compiler := jsonschema.NewCompiler()

	if r, err := os.Open("config/model/model_definition.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model_definition.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/model/model.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	var err error
	ModelDefJSONSchema, err = compiler.Compile("config/model/model_definition.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	ModelJSONSchema, err = compiler.Compile("config/model/model.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	modelJSONFile, err := os.ReadFile("config/model/model.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	if err := json.Unmarshal(modelJSONFile, &RegionHardwareJSON); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	taskInputJSONFile, err := os.ReadFile("config/model/task_input.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	if err := json.Unmarshal(taskInputJSONFile, &TaskInputJSON); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	taskOutputJSONFile, err := os.ReadFile("config/model/task_output.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	if err := json.Unmarshal(taskOutputJSONFile, &TaskOutputJSON); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
}

// ValidateJSONSchema validates the Protobuf message data
func ValidateJSONSchema(schema *jsonschema.Schema, msg any, emitUnpopulated bool) error {
	var v any
	var data []byte
	var err error
	switch msg := msg.(type) {
	case proto.Message:
		data, err = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: emitUnpopulated}.Marshal(msg)
		if err != nil {
			return err
		}
	default:
		data, err = json.Marshal(msg)
		if err != nil {
			return err
		}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if err := schema.Validate(v); err != nil {
		b, _ := json.MarshalIndent(err.(*jsonschema.ValidationError).DetailedOutput(), "", "  ")
		return errors.New(string(b))
	}

	return nil
}

// ValidateJSONSchemaString validates the string data given a string schema
func ValidateJSONSchemaString(schema *jsonschema.Schema, data string) error {
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return err
	}

	if err := schema.Validate(v); err != nil {
		return err
	}

	return nil
}
