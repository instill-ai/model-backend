package handler

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
// Note: id is now OUTPUT_ONLY (server-generated) after AIP refactoring
var immutableFields = []string{"model_definition", "configuration", "task", "region"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
// Updated for AIP Resource Refactoring - id is now server-generated
var outputOnlyFields = []string{
	"name", "id", "aliases",
	"create_time", "update_time", "delete_time",
	"permission", "input_schema", "output_schema",
	"versions", "stats",
	"owner_name", "owner", "creator_uid", "creator",
}

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
// var requiredFields = []string{"display_name", "model_definition", "configuration", "task", "visibility", "region", "hardware"}
