package handler

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"id", "model_definition", "configuration", "task", "region"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "ownerName", "owner", "create_time", "update_time", "delete_time"}

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
// var requiredFields = []string{"id", "model_definition", "configuration", "task", "visibility", "region", "hardware"}
