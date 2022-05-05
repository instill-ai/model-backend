package handler

import (
	"reflect"
	"regexp"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"Id"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"Id", "ModelDefinition", "Configuration"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"Name", "Uid", "Visibility", "Owner", "CreateTime", "UpdateTime"}

// Implementation follows https://google.aip.dev/203#required
func checkRequiredFields(pbModel *modelPB.Model) error {
	for _, field := range requiredFields {
		f := reflect.Indirect(reflect.ValueOf(pbModel)).FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			if f.String() == "" {
				return status.Errorf(codes.InvalidArgument, "Required field %s is not provided", field)
			}
		case reflect.Ptr:
			if f.IsNil() {
				return status.Errorf(codes.InvalidArgument, "Required field %s is not provided", field)
			}
		}
	}

	return nil
}

// Implementation follows https://google.aip.dev/203#output-only
func checkOutputOnlyFields(pbModel *modelPB.Model) error {
	for _, field := range outputOnlyFields {
		f := reflect.Indirect(reflect.ValueOf(pbModel)).FieldByName(field)
		switch f.Kind() {
		case reflect.Int32:
			reflect.ValueOf(pbModel).Elem().FieldByName(field).SetInt(0)
		case reflect.String:
			reflect.ValueOf(pbModel).Elem().FieldByName(field).SetString("")
		case reflect.Ptr:
			reflect.ValueOf(pbModel).Elem().FieldByName(field).Set(reflect.Zero(f.Type()))
		}
	}
	return nil
}

// Implementation follows https://google.aip.dev/203#immutable
func checkImmutableFields(pbModelReq *modelPB.Model, pbModelToUpdate *modelPB.Model) error {
	for _, field := range immutableFields {
		f := reflect.Indirect(reflect.ValueOf(pbModelReq)).FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			if f.String() != "" {
				if f.String() != reflect.Indirect(reflect.ValueOf(pbModelToUpdate)).FieldByName(field).String() {
					return status.Errorf(codes.InvalidArgument, "Field %s is immutable", field)
				}
			}
		}
	}
	return nil
}

// Implementation follows https://google.aip.dev/122#resource-id-segments
func checkResourceID(id string) error {
	if match, _ := regexp.MatchString("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$", id); !match {
		return status.Error(codes.InvalidArgument, "The id of Model needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
	}
	return nil
}
