package handler

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"id", "model_definition", "configuration"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "task", "state", "visibility", "ownerName", "owner", "create_time", "update_time", "delete_time"}
