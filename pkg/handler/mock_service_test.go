// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/pkg/service (interfaces: Service)

// Package handler_test is a generated GoMock package.
package handler_test

import (
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	modelv1alpha "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CreateModel mocks base method.
func (m *MockService) CreateModel(arg0 string, arg1 *datamodel.Model) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModel", arg0, arg1)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateModel indicates an expected call of CreateModel.
func (mr *MockServiceMockRecorder) CreateModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModel", reflect.TypeOf((*MockService)(nil).CreateModel), arg0, arg1)
}

// DeleteModel mocks base method.
func (m *MockService) DeleteModel(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModel", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModel indicates an expected call of DeleteModel.
func (mr *MockServiceMockRecorder) DeleteModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModel", reflect.TypeOf((*MockService)(nil).DeleteModel), arg0, arg1)
}

// DeployModelInstance mocks base method.
func (m *MockService) DeployModelInstance(arg0 uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeployModelInstance", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeployModelInstance indicates an expected call of DeployModelInstance.
func (mr *MockServiceMockRecorder) DeployModelInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeployModelInstance", reflect.TypeOf((*MockService)(nil).DeployModelInstance), arg0)
}

// GetModelById mocks base method.
func (m *MockService) GetModelById(arg0, arg1 string, arg2 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelById", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelById indicates an expected call of GetModelById.
func (mr *MockServiceMockRecorder) GetModelById(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelById", reflect.TypeOf((*MockService)(nil).GetModelById), arg0, arg1, arg2)
}

// GetModelByUid mocks base method.
func (m *MockService) GetModelByUid(arg0 string, arg1 uuid.UUID, arg2 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUid", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUid indicates an expected call of GetModelByUid.
func (mr *MockServiceMockRecorder) GetModelByUid(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUid", reflect.TypeOf((*MockService)(nil).GetModelByUid), arg0, arg1, arg2)
}

// GetModelDefinition mocks base method.
func (m *MockService) GetModelDefinition(arg0 string) (datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinition", arg0)
	ret0, _ := ret[0].(datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinition indicates an expected call of GetModelDefinition.
func (mr *MockServiceMockRecorder) GetModelDefinition(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinition", reflect.TypeOf((*MockService)(nil).GetModelDefinition), arg0)
}

// GetModelDefinitionByUid mocks base method.
func (m *MockService) GetModelDefinitionByUid(arg0 uuid.UUID) (datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinitionByUid", arg0)
	ret0, _ := ret[0].(datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinitionByUid indicates an expected call of GetModelDefinitionByUid.
func (mr *MockServiceMockRecorder) GetModelDefinitionByUid(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinitionByUid", reflect.TypeOf((*MockService)(nil).GetModelDefinitionByUid), arg0)
}

// GetModelInstance mocks base method.
func (m *MockService) GetModelInstance(arg0 uuid.UUID, arg1 string, arg2 modelv1alpha.View) (datamodel.ModelInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.ModelInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelInstance indicates an expected call of GetModelInstance.
func (mr *MockServiceMockRecorder) GetModelInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelInstance", reflect.TypeOf((*MockService)(nil).GetModelInstance), arg0, arg1, arg2)
}

// GetModelInstanceByUid mocks base method.
func (m *MockService) GetModelInstanceByUid(arg0, arg1 uuid.UUID, arg2 modelv1alpha.View) (datamodel.ModelInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelInstanceByUid", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.ModelInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelInstanceByUid indicates an expected call of GetModelInstanceByUid.
func (mr *MockServiceMockRecorder) GetModelInstanceByUid(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelInstanceByUid", reflect.TypeOf((*MockService)(nil).GetModelInstanceByUid), arg0, arg1, arg2)
}

// GetTritonEnsembleModel mocks base method.
func (m *MockService) GetTritonEnsembleModel(arg0 uuid.UUID) (datamodel.TritonModel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTritonEnsembleModel", arg0)
	ret0, _ := ret[0].(datamodel.TritonModel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTritonEnsembleModel indicates an expected call of GetTritonEnsembleModel.
func (mr *MockServiceMockRecorder) GetTritonEnsembleModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTritonEnsembleModel", reflect.TypeOf((*MockService)(nil).GetTritonEnsembleModel), arg0)
}

// GetTritonModels mocks base method.
func (m *MockService) GetTritonModels(arg0 uuid.UUID) ([]datamodel.TritonModel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTritonModels", arg0)
	ret0, _ := ret[0].([]datamodel.TritonModel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTritonModels indicates an expected call of GetTritonModels.
func (mr *MockServiceMockRecorder) GetTritonModels(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTritonModels", reflect.TypeOf((*MockService)(nil).GetTritonModels), arg0)
}

// ListModel mocks base method.
func (m *MockService) ListModel(arg0 string, arg1 modelv1alpha.View, arg2 int, arg3 string) ([]datamodel.Model, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModel", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]datamodel.Model)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModel indicates an expected call of ListModel.
func (mr *MockServiceMockRecorder) ListModel(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModel", reflect.TypeOf((*MockService)(nil).ListModel), arg0, arg1, arg2, arg3)
}

// ListModelDefinition mocks base method.
func (m *MockService) ListModelDefinition(arg0 modelv1alpha.View, arg1 int, arg2 string) ([]datamodel.ModelDefinition, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelDefinition", arg0, arg1, arg2)
	ret0, _ := ret[0].([]datamodel.ModelDefinition)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelDefinition indicates an expected call of ListModelDefinition.
func (mr *MockServiceMockRecorder) ListModelDefinition(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelDefinition", reflect.TypeOf((*MockService)(nil).ListModelDefinition), arg0, arg1, arg2)
}

// ListModelInstance mocks base method.
func (m *MockService) ListModelInstance(arg0 uuid.UUID, arg1 modelv1alpha.View, arg2 int, arg3 string) ([]datamodel.ModelInstance, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelInstance", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]datamodel.ModelInstance)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelInstance indicates an expected call of ListModelInstance.
func (mr *MockServiceMockRecorder) ListModelInstance(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelInstance", reflect.TypeOf((*MockService)(nil).ListModelInstance), arg0, arg1, arg2, arg3)
}

// ModelInfer mocks base method.
func (m *MockService) ModelInfer(arg0 uuid.UUID, arg1 [][]byte, arg2 modelv1alpha.ModelInstance_Task) ([]*modelv1alpha.ModelInstanceInferenceResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInfer", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*modelv1alpha.ModelInstanceInferenceResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelInfer indicates an expected call of ModelInfer.
func (mr *MockServiceMockRecorder) ModelInfer(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelInfer", reflect.TypeOf((*MockService)(nil).ModelInfer), arg0, arg1, arg2)
}

// ModelInferTestMode mocks base method.
func (m *MockService) ModelInferTestMode(arg0 string, arg1 uuid.UUID, arg2 [][]byte, arg3 modelv1alpha.ModelInstance_Task) ([]*modelv1alpha.ModelInstanceInferenceResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInferTestMode", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*modelv1alpha.ModelInstanceInferenceResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelInferTestMode indicates an expected call of ModelInferTestMode.
func (mr *MockServiceMockRecorder) ModelInferTestMode(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelInferTestMode", reflect.TypeOf((*MockService)(nil).ModelInferTestMode), arg0, arg1, arg2, arg3)
}

// PublishModel mocks base method.
func (m *MockService) PublishModel(arg0, arg1 string) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishModel", arg0, arg1)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PublishModel indicates an expected call of PublishModel.
func (mr *MockServiceMockRecorder) PublishModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishModel", reflect.TypeOf((*MockService)(nil).PublishModel), arg0, arg1)
}

// RenameModel mocks base method.
func (m *MockService) RenameModel(arg0, arg1, arg2 string) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RenameModel", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RenameModel indicates an expected call of RenameModel.
func (mr *MockServiceMockRecorder) RenameModel(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenameModel", reflect.TypeOf((*MockService)(nil).RenameModel), arg0, arg1, arg2)
}

// UndeployModelInstance mocks base method.
func (m *MockService) UndeployModelInstance(arg0 uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndeployModelInstance", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UndeployModelInstance indicates an expected call of UndeployModelInstance.
func (mr *MockServiceMockRecorder) UndeployModelInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndeployModelInstance", reflect.TypeOf((*MockService)(nil).UndeployModelInstance), arg0)
}

// UnpublishModel mocks base method.
func (m *MockService) UnpublishModel(arg0, arg1 string) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnpublishModel", arg0, arg1)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnpublishModel indicates an expected call of UnpublishModel.
func (mr *MockServiceMockRecorder) UnpublishModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnpublishModel", reflect.TypeOf((*MockService)(nil).UnpublishModel), arg0, arg1)
}

// UpdateModel mocks base method.
func (m *MockService) UpdateModel(arg0 uuid.UUID, arg1 *datamodel.Model) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModel", arg0, arg1)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateModel indicates an expected call of UpdateModel.
func (mr *MockServiceMockRecorder) UpdateModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModel", reflect.TypeOf((*MockService)(nil).UpdateModel), arg0, arg1)
}
