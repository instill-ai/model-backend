package datamodel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/lestrrat-go/jsref/provider"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/jsonref"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
)

// ModelDefJSONSchema represents the ModelDefinition JSON Schema for validating the payload
var ModelDefJSONSchema *jsonschema.Schema

// ModelJSONSchema represents the Model JSON Schema for validating the payload
var ModelJSONSchema *jsonschema.Schema

var RegionHardwareJSON RegionHardware

// Tasks schema
var TasksJSONMap map[string]map[string]any
var TasksJSONInputSchemaMap map[string]*jsonschema.Schema
var TasksJSONOutputSchemaMap map[string]*jsonschema.Schema

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

func renderJSON(tasksJSONBytes []byte) ([]byte, error) {
	var err error
	res := jsonref.New()
	err = res.AddProvider(provider.NewHTTP())
	if err != nil {
		return nil, err
	}

	var tasksJSON any
	err = json.Unmarshal(tasksJSONBytes, &tasksJSON)
	if err != nil {
		return nil, err
	}

	result, err := res.Resolve(tasksJSON, "", jsonref.WithRecursiveResolution(true))
	if err != nil {
		return nil, err
	}
	renderedTasksJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return renderedTasksJSON, nil

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

	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/instill-ai/instill-core/%s/schema/ai-tasks.json", config.Config.Server.TaskSchemaVersion))
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Fatal(fmt.Sprintf("failed to fetch data: %s", resp.Status))
	}

	TasksJSONMap = map[string]map[string]any{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	renderedSchemaBytes, err := renderJSON(body)
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
	err = json.Unmarshal(renderedSchemaBytes, &TasksJSONMap)
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	TasksJSONInputSchemaMap = map[string]*jsonschema.Schema{}
	TasksJSONOutputSchemaMap = map[string]*jsonschema.Schema{}
	for task := range TasksJSONMap {
		inputSchemaBytes, err := json.Marshal(TasksJSONMap[task]["input"])
		if err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
		if err = compiler.AddResource(fmt.Sprintf("%v_INPUT.json", task), bytes.NewReader(inputSchemaBytes)); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
		if TasksJSONInputSchemaMap[task], err = compiler.Compile(fmt.Sprintf("%v_INPUT.json", task)); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
		outputSchemaBytes, err := json.Marshal(TasksJSONMap[task]["output"])
		if err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
		if err = compiler.AddResource(fmt.Sprintf("%v_OUTPUT.json", task), bytes.NewReader(outputSchemaBytes)); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
		if TasksJSONOutputSchemaMap[task], err = compiler.Compile(fmt.Sprintf("%v_OUTPUT.json", task)); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
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
