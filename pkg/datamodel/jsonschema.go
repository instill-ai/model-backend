package datamodel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/model-backend/pkg/logger"
)

// ModelDefJSONSchema represents the ModelDefinition JSON Schema for validating the payload
var ModelDefJSONSchema *jsonschema.Schema

// ModelJSONSchema represents the Model JSON Schema for validating the payload
var ModelJSONSchema *jsonschema.Schema

// ModelCardJSONSchema represents the Model Instance Card JSON Schema for validating the payload
var ModelCardJSONSchema *jsonschema.Schema

// GCSUserAccountJSONSchema represents the GCS User Account JSON Schema for validating the payload
var GCSUserAccountJSONSchema *jsonschema.Schema

// GCSServiceAccountJSONSchema represents the GCS Service Account JSON Schema for validating the payload
var GCSServiceAccountJSONSchema *jsonschema.Schema

// InitJSONSchema initialise JSON Schema instances with the given files
func InitJSONSchema(ctx context.Context) {

	logger, _ := logger.GetZapLogger(ctx)

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

	if r, err := os.Open("config/model/model_spec.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model_spec.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/model/model_card.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/model/model_card.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/credential/gcs_user_account.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/credential/gcs_user_account.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/credential/gcs_service_account.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/model-backend/blob/main/config/credential/gcs_service_account.json", r); err != nil {
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

	ModelCardJSONSchema, err = compiler.Compile("config/model/model_card.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	GCSUserAccountJSONSchema, err = compiler.Compile("config/credential/gcs_user_account.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	GCSServiceAccountJSONSchema, err = compiler.Compile("config/credential/gcs_service_account.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}
}

// ValidateJSONSchema validates the Protobuf message data
func ValidateJSONSchema(schema *jsonschema.Schema, msg interface{}, emitUnpopulated bool) error {
	var v interface{}
	var data []byte
	var err error
	switch msg := msg.(type) {
	case proto.Message:
		data, err = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: emitUnpopulated}.Marshal(msg)
		if err != nil {
			return err
		}
	case LocalModelConfiguration:
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
		return fmt.Errorf(string(b))
	}

	return nil
}

// ValidateJSONSchemaString validates the string data given a string schema
func ValidateJSONSchemaString(schema *jsonschema.Schema, data string) error {
	var v interface{}
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return err
	}

	if err := schema.Validate(v); err != nil {
		return err
	}

	return nil
}
