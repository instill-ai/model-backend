package datamodel

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/model-backend/internal/logger"
)

// ModelDefJSONSchema represents the ModelDefinition JSON Schema for validating the payload
var ModelDefJSONSchema *jsonschema.Schema

// ModelJSONSchema represents the Model JSON Schema for validating the payload
var ModelJSONSchema *jsonschema.Schema

// ModelInstanceJSONSchema represents the Model Instance JSON Schema for validating the payload
var ModelInstanceJSONSchema *jsonschema.Schema

// ModelInstanceCardJSONSchema represents the Model Instance Card JSON Schema for validating the payload
var ModelInstanceCardJSONSchema *jsonschema.Schema

// InitJSONSchema initialise JSON Schema instances with the given files
func InitJSONSchema() {

	logger, _ := logger.GetZapLogger()

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

	if r, err := os.Open("config/model/model_instance.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model_instance.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/model/model_instance_card.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model_instance_card.json", r); err != nil {
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

	ModelInstanceJSONSchema, err = compiler.Compile("config/model/model_instance.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	ModelInstanceCardJSONSchema, err = compiler.Compile("config/model/model_instance_card.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

}

//ValidateJSONSchema validates the Protobuf message data
func ValidateJSONSchema(schema *jsonschema.Schema, msg interface{}, emitUnpopulated bool) error {
	data, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: emitUnpopulated}.Marshal(msg.(proto.Message))
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if err := schema.Validate(v); err != nil {
		b, _ := json.MarshalIndent(err.(*jsonschema.ValidationError).DetailedOutput(), "", "  ")
		return fmt.Errorf(string(b))
	}

	return nil
}

// ValidateJSONSchemaString validates the string data given a string schema
func ValidateJSONSchemaString(schema string, data string) error {

	sch, err := jsonschema.CompileString("schema.json", schema)
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return err
	}

	if err = sch.Validate(v); err != nil {
		return err
	}

	return nil
}
