// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/internal/triton (interfaces: Triton)
// Package service_test is a generated GoMock package.
package service_test

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	inferenceserver "github.com/instill-ai/model-backend/internal/inferenceserver"
	modelv1alpha "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// MockTriton is a mock of Triton interface.
type MockTriton struct {
	ctrl     *gomock.Controller
	recorder *MockTritonMockRecorder
}

// MockTritonMockRecorder is the mock recorder for MockTriton.
type MockTritonMockRecorder struct {
	mock *MockTriton
}

// NewMockTriton creates a new mock instance.
func NewMockTriton(ctrl *gomock.Controller) *MockTriton {
	mock := &MockTriton{ctrl: ctrl}
	mock.recorder = &MockTritonMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTriton) EXPECT() *MockTritonMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockTriton) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockTritonMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockTriton)(nil).Close))
}

// Init mocks base method.
func (m *MockTriton) Init() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Init")
}

// Init indicates an expected call of Init.
func (mr *MockTritonMockRecorder) Init() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockTriton)(nil).Init))
}

// IsTritonServerReady mocks base method.
func (m *MockTriton) IsTritonServerReady() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsTritonServerReady")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsTritonServerReady indicates an expected call of IsTritonServerReady.
func (mr *MockTritonMockRecorder) IsTritonServerReady() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsTritonServerReady", reflect.TypeOf((*MockTriton)(nil).IsTritonServerReady))
}

// ListModelsRequest mocks base method.
func (m *MockTriton) ListModelsRequest() *inferenceserver.RepositoryIndexResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelsRequest")
	ret0, _ := ret[0].(*inferenceserver.RepositoryIndexResponse)
	return ret0
}

// ListModelsRequest indicates an expected call of ListModelsRequest.
func (mr *MockTritonMockRecorder) ListModelsRequest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelsRequest", reflect.TypeOf((*MockTriton)(nil).ListModelsRequest))
}

// LoadModelRequest mocks base method.
func (m *MockTriton) LoadModelRequest(arg0 string) (*inferenceserver.RepositoryModelLoadResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadModelRequest", arg0)
	ret0, _ := ret[0].(*inferenceserver.RepositoryModelLoadResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadModelRequest indicates an expected call of LoadModelRequest.
func (mr *MockTritonMockRecorder) LoadModelRequest(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadModelRequest", reflect.TypeOf((*MockTriton)(nil).LoadModelRequest), arg0)
}

// ModelConfigRequest mocks base method.
func (m *MockTriton) ModelConfigRequest(arg0, arg1 string) *inferenceserver.ModelConfigResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelConfigRequest", arg0, arg1)
	ret0, _ := ret[0].(*inferenceserver.ModelConfigResponse)
	return ret0
}

// ModelConfigRequest indicates an expected call of ModelConfigRequest.
func (mr *MockTritonMockRecorder) ModelConfigRequest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelConfigRequest", reflect.TypeOf((*MockTriton)(nil).ModelConfigRequest), arg0, arg1)
}

// ModelInferRequest mocks base method.
func (m *MockTriton) ModelInferRequest(arg0 modelv1alpha.ModelInstance_Task, arg1 [][]byte, arg2, arg3 string, arg4 *inferenceserver.ModelMetadataResponse, arg5 *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInferRequest", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(*inferenceserver.ModelInferResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelInferRequest indicates an expected call of ModelInferRequest.
func (mr *MockTritonMockRecorder) ModelInferRequest(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelInferRequest", reflect.TypeOf((*MockTriton)(nil).ModelInferRequest), arg0, arg1, arg2, arg3, arg4, arg5)
}

// ModelMetadataRequest mocks base method.
func (m *MockTriton) ModelMetadataRequest(arg0, arg1 string) *inferenceserver.ModelMetadataResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelMetadataRequest", arg0, arg1)
	ret0, _ := ret[0].(*inferenceserver.ModelMetadataResponse)
	return ret0
}

// ModelMetadataRequest indicates an expected call of ModelMetadataRequest.
func (mr *MockTritonMockRecorder) ModelMetadataRequest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelMetadataRequest", reflect.TypeOf((*MockTriton)(nil).ModelMetadataRequest), arg0, arg1)
}

// PostProcess mocks base method.
func (m *MockTriton) PostProcess(arg0 *inferenceserver.ModelInferResponse, arg1 *inferenceserver.ModelMetadataResponse, arg2 modelv1alpha.ModelInstance_Task) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostProcess", arg0, arg1, arg2)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PostProcess indicates an expected call of PostProcess.
func (mr *MockTritonMockRecorder) PostProcess(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostProcess", reflect.TypeOf((*MockTriton)(nil).PostProcess), arg0, arg1, arg2)
}

// ServerLiveRequest mocks base method.
func (m *MockTriton) ServerLiveRequest() *inferenceserver.ServerLiveResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerLiveRequest")
	ret0, _ := ret[0].(*inferenceserver.ServerLiveResponse)
	return ret0
}

// ServerLiveRequest indicates an expected call of ServerLiveRequest.
func (mr *MockTritonMockRecorder) ServerLiveRequest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerLiveRequest", reflect.TypeOf((*MockTriton)(nil).ServerLiveRequest))
}

// ServerReadyRequest mocks base method.
func (m *MockTriton) ServerReadyRequest() *inferenceserver.ServerReadyResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerReadyRequest")
	ret0, _ := ret[0].(*inferenceserver.ServerReadyResponse)
	return ret0
}

// ServerReadyRequest indicates an expected call of ServerReadyRequest.
func (mr *MockTritonMockRecorder) ServerReadyRequest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerReadyRequest", reflect.TypeOf((*MockTriton)(nil).ServerReadyRequest))
}

// UnloadModelRequest mocks base method.
func (m *MockTriton) UnloadModelRequest(arg0 string) (*inferenceserver.RepositoryModelUnloadResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnloadModelRequest", arg0)
	ret0, _ := ret[0].(*inferenceserver.RepositoryModelUnloadResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnloadModelRequest indicates an expected call of UnloadModelRequest.
func (mr *MockTritonMockRecorder) UnloadModelRequest(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnloadModelRequest", reflect.TypeOf((*MockTriton)(nil).UnloadModelRequest), arg0)
}
