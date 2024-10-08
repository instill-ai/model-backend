// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: ray.proto

package rayserver

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// CallRequest represents a request for model inference
type CallRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Inference input parameters.
	TaskInputs []*structpb.Struct `protobuf:"bytes,1,rep,name=task_inputs,json=taskInputs,proto3" json:"task_inputs,omitempty"`
}

func (x *CallRequest) Reset() {
	*x = CallRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ray_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CallRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CallRequest) ProtoMessage() {}

func (x *CallRequest) ProtoReflect() protoreflect.Message {
	mi := &file_ray_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CallRequest.ProtoReflect.Descriptor instead.
func (*CallRequest) Descriptor() ([]byte, []int) {
	return file_ray_proto_rawDescGZIP(), []int{0}
}

func (x *CallRequest) GetTaskInputs() []*structpb.Struct {
	if x != nil {
		return x.TaskInputs
	}
	return nil
}

// CallResponse represents a response for model inference
type CallResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Model inference outputs.
	TaskOutputs []*structpb.Struct `protobuf:"bytes,1,rep,name=task_outputs,json=taskOutputs,proto3" json:"task_outputs,omitempty"`
}

func (x *CallResponse) Reset() {
	*x = CallResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ray_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CallResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CallResponse) ProtoMessage() {}

func (x *CallResponse) ProtoReflect() protoreflect.Message {
	mi := &file_ray_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CallResponse.ProtoReflect.Descriptor instead.
func (*CallResponse) Descriptor() ([]byte, []int) {
	return file_ray_proto_rawDescGZIP(), []int{1}
}

func (x *CallResponse) GetTaskOutputs() []*structpb.Struct {
	if x != nil {
		return x.TaskOutputs
	}
	return nil
}

var File_ray_proto protoreflect.FileDescriptor

var file_ray_proto_rawDesc = []byte{
	0x0a, 0x09, 0x72, 0x61, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x72, 0x61, 0x79,
	0x2e, 0x76, 0x31, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x47, 0x0a, 0x0b, 0x43, 0x61, 0x6c, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x38, 0x0a, 0x0b, 0x74, 0x61, 0x73, 0x6b, 0x5f, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x0a,
	0x74, 0x61, 0x73, 0x6b, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x22, 0x4a, 0x0a, 0x0c, 0x43, 0x61,
	0x6c, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3a, 0x0a, 0x0c, 0x74, 0x61,
	0x73, 0x6b, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x0b, 0x74, 0x61, 0x73, 0x6b, 0x4f,
	0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x32, 0x43, 0x0a, 0x0a, 0x52, 0x61, 0x79, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x35, 0x0a, 0x08, 0x5f, 0x5f, 0x63, 0x61, 0x6c, 0x6c, 0x5f, 0x5f,
	0x12, 0x13, 0x2e, 0x72, 0x61, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x61, 0x6c, 0x6c, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x14, 0x2e, 0x72, 0x61, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x43,
	0x61, 0x6c, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x0d, 0x5a, 0x0b, 0x2e,
	0x2f, 0x72, 0x61, 0x79, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_ray_proto_rawDescOnce sync.Once
	file_ray_proto_rawDescData = file_ray_proto_rawDesc
)

func file_ray_proto_rawDescGZIP() []byte {
	file_ray_proto_rawDescOnce.Do(func() {
		file_ray_proto_rawDescData = protoimpl.X.CompressGZIP(file_ray_proto_rawDescData)
	})
	return file_ray_proto_rawDescData
}

var file_ray_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_ray_proto_goTypes = []interface{}{
	(*CallRequest)(nil),     // 0: ray.v1.CallRequest
	(*CallResponse)(nil),    // 1: ray.v1.CallResponse
	(*structpb.Struct)(nil), // 2: google.protobuf.Struct
}
var file_ray_proto_depIdxs = []int32{
	2, // 0: ray.v1.CallRequest.task_inputs:type_name -> google.protobuf.Struct
	2, // 1: ray.v1.CallResponse.task_outputs:type_name -> google.protobuf.Struct
	0, // 2: ray.v1.RayService.__call__:input_type -> ray.v1.CallRequest
	1, // 3: ray.v1.RayService.__call__:output_type -> ray.v1.CallResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_ray_proto_init() }
func file_ray_proto_init() {
	if File_ray_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ray_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CallRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_ray_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CallResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_ray_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_ray_proto_goTypes,
		DependencyIndexes: file_ray_proto_depIdxs,
		MessageInfos:      file_ray_proto_msgTypes,
	}.Build()
	File_ray_proto = out.File
	file_ray_proto_rawDesc = nil
	file_ray_proto_goTypes = nil
	file_ray_proto_depIdxs = nil
}
