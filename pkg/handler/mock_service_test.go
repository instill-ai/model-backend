// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/pkg/service (interfaces: Service)

// Package handler_test is a generated GoMock package.
package handler_test

import (
	reflect "reflect"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	worker "github.com/instill-ai/model-backend/internal/worker"
	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	service "github.com/instill-ai/model-backend/pkg/service"
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

// CancelOperation mocks base method.
func (m *MockService) CancelOperation(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelOperation", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CancelOperation indicates an expected call of CancelOperation.
func (mr *MockServiceMockRecorder) CancelOperation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelOperation", reflect.TypeOf((*MockService)(nil).CancelOperation), arg0)
}

// CreateModelAsync mocks base method.
func (m *MockService) CreateModelAsync(arg0 string, arg1 *datamodel.Model) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelAsync", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateModelAsync indicates an expected call of CreateModelAsync.
func (mr *MockServiceMockRecorder) CreateModelAsync(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelAsync", reflect.TypeOf((*MockService)(nil).CreateModelAsync), arg0, arg1)
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

// DeployModelInstanceAsync mocks base method.
func (m *MockService) DeployModelInstanceAsync(arg0 string, arg1, arg2 uuid.UUID) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeployModelInstanceAsync", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeployModelInstanceAsync indicates an expected call of DeployModelInstanceAsync.
func (mr *MockServiceMockRecorder) DeployModelInstanceAsync(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeployModelInstanceAsync", reflect.TypeOf((*MockService)(nil).DeployModelInstanceAsync), arg0, arg1, arg2)
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

// GetOperation mocks base method.
func (m *MockService) GetOperation(arg0 string) (*longrunningpb.Operation, *worker.ModelInstanceParams, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperation", arg0)
	ret0, _ := ret[0].(*longrunningpb.Operation)
	ret1, _ := ret[1].(*worker.ModelInstanceParams)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// GetOperation indicates an expected call of GetOperation.
func (mr *MockServiceMockRecorder) GetOperation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperation", reflect.TypeOf((*MockService)(nil).GetOperation), arg0)
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

// ListModelDefinitions mocks base method.
func (m *MockService) ListModelDefinitions(arg0 modelv1alpha.View, arg1 int, arg2 string) ([]datamodel.ModelDefinition, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelDefinitions", arg0, arg1, arg2)
	ret0, _ := ret[0].([]datamodel.ModelDefinition)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelDefinitions indicates an expected call of ListModelDefinitions.
func (mr *MockServiceMockRecorder) ListModelDefinitions(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelDefinitions", reflect.TypeOf((*MockService)(nil).ListModelDefinitions), arg0, arg1, arg2)
}

// ListModelInstances mocks base method.
func (m *MockService) ListModelInstances(arg0 uuid.UUID, arg1 modelv1alpha.View, arg2 int, arg3 string) ([]datamodel.ModelInstance, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelInstances", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]datamodel.ModelInstance)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelInstances indicates an expected call of ListModelInstances.
func (mr *MockServiceMockRecorder) ListModelInstances(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelInstances", reflect.TypeOf((*MockService)(nil).ListModelInstances), arg0, arg1, arg2, arg3)
}

// ListModels mocks base method.
func (m *MockService) ListModels(arg0 string, arg1 modelv1alpha.View, arg2 int, arg3 string) ([]datamodel.Model, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModels", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]datamodel.Model)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModels indicates an expected call of ListModels.
func (mr *MockServiceMockRecorder) ListModels(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModels", reflect.TypeOf((*MockService)(nil).ListModels), arg0, arg1, arg2, arg3)
}

// ListOperation mocks base method.
func (m *MockService) ListOperation(arg0 int, arg1 string) ([]*longrunningpb.Operation, []*worker.ModelInstanceParams, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperation", arg0, arg1)
	ret0, _ := ret[0].([]*longrunningpb.Operation)
	ret1, _ := ret[1].([]*worker.ModelInstanceParams)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(int64)
	ret4, _ := ret[4].(error)
	return ret0, ret1, ret2, ret3, ret4
}

// ListOperation indicates an expected call of ListOperation.
func (mr *MockServiceMockRecorder) ListOperation(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperation", reflect.TypeOf((*MockService)(nil).ListOperation), arg0, arg1)
}

// ModelInfer mocks base method.
func (m *MockService) ModelInfer(arg0 uuid.UUID, arg1 service.InferInput, arg2 modelv1alpha.ModelInstance_Task) ([]*modelv1alpha.TaskOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInfer", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*modelv1alpha.TaskOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelInfer indicates an expected call of ModelInfer.
func (mr *MockServiceMockRecorder) ModelInfer(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelInfer", reflect.TypeOf((*MockService)(nil).ModelInfer), arg0, arg1, arg2)
}

// ModelInferTestMode mocks base method.
func (m *MockService) ModelInferTestMode(arg0 string, arg1 uuid.UUID, arg2 service.InferInput, arg3 modelv1alpha.ModelInstance_Task) ([]*modelv1alpha.TaskOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInferTestMode", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*modelv1alpha.TaskOutput)
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

// SearchAttributeReady mocks base method.
func (m *MockService) SearchAttributeReady() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SearchAttributeReady")
	ret0, _ := ret[0].(error)
	return ret0
}

// SearchAttributeReady indicates an expected call of SearchAttributeReady.
func (mr *MockServiceMockRecorder) SearchAttributeReady() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchAttributeReady", reflect.TypeOf((*MockService)(nil).SearchAttributeReady))
}

// UndeployModelInstanceAsync mocks base method.
func (m *MockService) UndeployModelInstanceAsync(arg0 string, arg1, arg2 uuid.UUID) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndeployModelInstanceAsync", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UndeployModelInstanceAsync indicates an expected call of UndeployModelInstanceAsync.
func (mr *MockServiceMockRecorder) UndeployModelInstanceAsync(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndeployModelInstanceAsync", reflect.TypeOf((*MockService)(nil).UndeployModelInstanceAsync), arg0, arg1, arg2)
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

// UpdateModelInstance mocks base method.
func (m *MockService) UpdateModelInstance(arg0 uuid.UUID, arg1 datamodel.ModelInstance) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModelInstance", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModelInstance indicates an expected call of UpdateModelInstance.
func (mr *MockServiceMockRecorder) UpdateModelInstance(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelInstance", reflect.TypeOf((*MockService)(nil).UpdateModelInstance), arg0, arg1)
}
