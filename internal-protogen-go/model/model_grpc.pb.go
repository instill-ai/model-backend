// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package model

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ModelClient is the client API for Model service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ModelClient interface {
	CreateModel(ctx context.Context, opts ...grpc.CallOption) (Model_CreateModelClient, error)
	LoadModel(ctx context.Context, in *LoadModelRequest, opts ...grpc.CallOption) (*LoadModelResponse, error)
	UnloadModel(ctx context.Context, in *UnloadModelRequest, opts ...grpc.CallOption) (*UnloadModelResponse, error)
	PredictModel(ctx context.Context, in *PredictModelRequest, opts ...grpc.CallOption) (*PredictModelResponse, error)
	ListModels(ctx context.Context, in *ListModelRequest, opts ...grpc.CallOption) (*ListModelResponse, error)
}

type modelClient struct {
	cc grpc.ClientConnInterface
}

func NewModelClient(cc grpc.ClientConnInterface) ModelClient {
	return &modelClient{cc}
}

func (c *modelClient) CreateModel(ctx context.Context, opts ...grpc.CallOption) (Model_CreateModelClient, error) {
	stream, err := c.cc.NewStream(ctx, &Model_ServiceDesc.Streams[0], "/instill.model.Model/createModel", opts...)
	if err != nil {
		return nil, err
	}
	x := &modelCreateModelClient{stream}
	return x, nil
}

type Model_CreateModelClient interface {
	Send(*CreateModelRequest) error
	CloseAndRecv() (*CreateModelResponse, error)
	grpc.ClientStream
}

type modelCreateModelClient struct {
	grpc.ClientStream
}

func (x *modelCreateModelClient) Send(m *CreateModelRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *modelCreateModelClient) CloseAndRecv() (*CreateModelResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(CreateModelResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *modelClient) LoadModel(ctx context.Context, in *LoadModelRequest, opts ...grpc.CallOption) (*LoadModelResponse, error) {
	out := new(LoadModelResponse)
	err := c.cc.Invoke(ctx, "/instill.model.Model/loadModel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *modelClient) UnloadModel(ctx context.Context, in *UnloadModelRequest, opts ...grpc.CallOption) (*UnloadModelResponse, error) {
	out := new(UnloadModelResponse)
	err := c.cc.Invoke(ctx, "/instill.model.Model/unloadModel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *modelClient) PredictModel(ctx context.Context, in *PredictModelRequest, opts ...grpc.CallOption) (*PredictModelResponse, error) {
	out := new(PredictModelResponse)
	err := c.cc.Invoke(ctx, "/instill.model.Model/predictModel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *modelClient) ListModels(ctx context.Context, in *ListModelRequest, opts ...grpc.CallOption) (*ListModelResponse, error) {
	out := new(ListModelResponse)
	err := c.cc.Invoke(ctx, "/instill.model.Model/listModels", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ModelServer is the server API for Model service.
// All implementations should embed UnimplementedModelServer
// for forward compatibility
type ModelServer interface {
	CreateModel(Model_CreateModelServer) error
	LoadModel(context.Context, *LoadModelRequest) (*LoadModelResponse, error)
	UnloadModel(context.Context, *UnloadModelRequest) (*UnloadModelResponse, error)
	PredictModel(context.Context, *PredictModelRequest) (*PredictModelResponse, error)
	ListModels(context.Context, *ListModelRequest) (*ListModelResponse, error)
}

// UnimplementedModelServer should be embedded to have forward compatible implementations.
type UnimplementedModelServer struct {
}

func (UnimplementedModelServer) CreateModel(Model_CreateModelServer) error {
	return status.Errorf(codes.Unimplemented, "method CreateModel not implemented")
}
func (UnimplementedModelServer) LoadModel(context.Context, *LoadModelRequest) (*LoadModelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoadModel not implemented")
}
func (UnimplementedModelServer) UnloadModel(context.Context, *UnloadModelRequest) (*UnloadModelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnloadModel not implemented")
}
func (UnimplementedModelServer) PredictModel(context.Context, *PredictModelRequest) (*PredictModelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PredictModel not implemented")
}
func (UnimplementedModelServer) ListModels(context.Context, *ListModelRequest) (*ListModelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListModels not implemented")
}

// UnsafeModelServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ModelServer will
// result in compilation errors.
type UnsafeModelServer interface {
	mustEmbedUnimplementedModelServer()
}

func RegisterModelServer(s grpc.ServiceRegistrar, srv ModelServer) {
	s.RegisterService(&Model_ServiceDesc, srv)
}

func _Model_CreateModel_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ModelServer).CreateModel(&modelCreateModelServer{stream})
}

type Model_CreateModelServer interface {
	SendAndClose(*CreateModelResponse) error
	Recv() (*CreateModelRequest, error)
	grpc.ServerStream
}

type modelCreateModelServer struct {
	grpc.ServerStream
}

func (x *modelCreateModelServer) SendAndClose(m *CreateModelResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *modelCreateModelServer) Recv() (*CreateModelRequest, error) {
	m := new(CreateModelRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Model_LoadModel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoadModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModelServer).LoadModel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/instill.model.Model/loadModel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModelServer).LoadModel(ctx, req.(*LoadModelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Model_UnloadModel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnloadModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModelServer).UnloadModel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/instill.model.Model/unloadModel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModelServer).UnloadModel(ctx, req.(*UnloadModelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Model_PredictModel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PredictModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModelServer).PredictModel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/instill.model.Model/predictModel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModelServer).PredictModel(ctx, req.(*PredictModelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Model_ListModels_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModelServer).ListModels(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/instill.model.Model/listModels",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModelServer).ListModels(ctx, req.(*ListModelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Model_ServiceDesc is the grpc.ServiceDesc for Model service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Model_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "instill.model.Model",
	HandlerType: (*ModelServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "loadModel",
			Handler:    _Model_LoadModel_Handler,
		},
		{
			MethodName: "unloadModel",
			Handler:    _Model_UnloadModel_Handler,
		},
		{
			MethodName: "predictModel",
			Handler:    _Model_PredictModel_Handler,
		},
		{
			MethodName: "listModels",
			Handler:    _Model_ListModels_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "createModel",
			Handler:       _Model_CreateModel_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "model/model.proto",
}
