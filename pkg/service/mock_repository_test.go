// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/pkg/repository (interfaces: Repository)

// Package service_test is a generated GoMock package.
package service_test

import (
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	modelv1alpha "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// CreateModel mocks base method.
func (m *MockRepository) CreateModel(arg0 datamodel.Model) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModel", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateModel indicates an expected call of CreateModel.
func (mr *MockRepositoryMockRecorder) CreateModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModel", reflect.TypeOf((*MockRepository)(nil).CreateModel), arg0)
}

// CreateTritonModel mocks base method.
func (m *MockRepository) CreateTritonModel(arg0 datamodel.TritonModel) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTritonModel", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTritonModel indicates an expected call of CreateTritonModel.
func (mr *MockRepositoryMockRecorder) CreateTritonModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTritonModel", reflect.TypeOf((*MockRepository)(nil).CreateTritonModel), arg0)
}

// DeleteModel mocks base method.
func (m *MockRepository) DeleteModel(arg0 uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModel", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModel indicates an expected call of DeleteModel.
func (mr *MockRepositoryMockRecorder) DeleteModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModel", reflect.TypeOf((*MockRepository)(nil).DeleteModel), arg0)
}

// GetModelById mocks base method.
func (m *MockRepository) GetModelById(arg0, arg1 string, arg2 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelById", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelById indicates an expected call of GetModelById.
func (mr *MockRepositoryMockRecorder) GetModelById(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelById", reflect.TypeOf((*MockRepository)(nil).GetModelById), arg0, arg1, arg2)
}

// GetModelByIdAdmin mocks base method.
func (m *MockRepository) GetModelByIdAdmin(arg0 string, arg1 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByIdAdmin", arg0, arg1)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByIdAdmin indicates an expected call of GetModelByIdAdmin.
func (mr *MockRepositoryMockRecorder) GetModelByIdAdmin(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByIdAdmin", reflect.TypeOf((*MockRepository)(nil).GetModelByIdAdmin), arg0, arg1)
}

// GetModelByUid mocks base method.
func (m *MockRepository) GetModelByUid(arg0 string, arg1 uuid.UUID, arg2 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUid", arg0, arg1, arg2)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUid indicates an expected call of GetModelByUid.
func (mr *MockRepositoryMockRecorder) GetModelByUid(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUid", reflect.TypeOf((*MockRepository)(nil).GetModelByUid), arg0, arg1, arg2)
}

// GetModelByUidAdmin mocks base method.
func (m *MockRepository) GetModelByUidAdmin(arg0 uuid.UUID, arg1 modelv1alpha.View) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUidAdmin", arg0, arg1)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUidAdmin indicates an expected call of GetModelByUidAdmin.
func (mr *MockRepositoryMockRecorder) GetModelByUidAdmin(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUidAdmin", reflect.TypeOf((*MockRepository)(nil).GetModelByUidAdmin), arg0, arg1)
}

// GetModelDefinition mocks base method.
func (m *MockRepository) GetModelDefinition(arg0 string) (datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinition", arg0)
	ret0, _ := ret[0].(datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinition indicates an expected call of GetModelDefinition.
func (mr *MockRepositoryMockRecorder) GetModelDefinition(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinition", reflect.TypeOf((*MockRepository)(nil).GetModelDefinition), arg0)
}

// GetModelDefinitionByUid mocks base method.
func (m *MockRepository) GetModelDefinitionByUid(arg0 uuid.UUID) (datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinitionByUid", arg0)
	ret0, _ := ret[0].(datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinitionByUid indicates an expected call of GetModelDefinitionByUid.
func (mr *MockRepositoryMockRecorder) GetModelDefinitionByUid(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinitionByUid", reflect.TypeOf((*MockRepository)(nil).GetModelDefinitionByUid), arg0)
}

// GetTritonEnsembleModel mocks base method.
func (m *MockRepository) GetTritonEnsembleModel(arg0 uuid.UUID) (datamodel.TritonModel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTritonEnsembleModel", arg0)
	ret0, _ := ret[0].(datamodel.TritonModel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTritonEnsembleModel indicates an expected call of GetTritonEnsembleModel.
func (mr *MockRepositoryMockRecorder) GetTritonEnsembleModel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTritonEnsembleModel", reflect.TypeOf((*MockRepository)(nil).GetTritonEnsembleModel), arg0)
}

// GetTritonModels mocks base method.
func (m *MockRepository) GetTritonModels(arg0 uuid.UUID) ([]datamodel.TritonModel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTritonModels", arg0)
	ret0, _ := ret[0].([]datamodel.TritonModel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTritonModels indicates an expected call of GetTritonModels.
func (mr *MockRepositoryMockRecorder) GetTritonModels(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTritonModels", reflect.TypeOf((*MockRepository)(nil).GetTritonModels), arg0)
}

// ListModelDefinitions mocks base method.
func (m *MockRepository) ListModelDefinitions(arg0 modelv1alpha.View, arg1 int, arg2 string) ([]datamodel.ModelDefinition, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelDefinitions", arg0, arg1, arg2)
	ret0, _ := ret[0].([]datamodel.ModelDefinition)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelDefinitions indicates an expected call of ListModelDefinitions.
func (mr *MockRepositoryMockRecorder) ListModelDefinitions(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelDefinitions", reflect.TypeOf((*MockRepository)(nil).ListModelDefinitions), arg0, arg1, arg2)
}

// ListModels mocks base method.
func (m *MockRepository) ListModels(arg0 string, arg1 modelv1alpha.View, arg2 int, arg3 string) ([]datamodel.Model, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModels", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]datamodel.Model)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModels indicates an expected call of ListModels.
func (mr *MockRepositoryMockRecorder) ListModels(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModels", reflect.TypeOf((*MockRepository)(nil).ListModels), arg0, arg1, arg2, arg3)
}

// ListModelsAdmin mocks base method.
func (m *MockRepository) ListModelsAdmin(arg0 modelv1alpha.View, arg1 int, arg2 string) ([]datamodel.Model, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelsAdmin", arg0, arg1, arg2)
	ret0, _ := ret[0].([]datamodel.Model)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelsAdmin indicates an expected call of ListModelsAdmin.
func (mr *MockRepositoryMockRecorder) ListModelsAdmin(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelsAdmin", reflect.TypeOf((*MockRepository)(nil).ListModelsAdmin), arg0, arg1, arg2)
}

// UpdateModel mocks base method.
func (m *MockRepository) UpdateModel(arg0 uuid.UUID, arg1 datamodel.Model) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModel", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModel indicates an expected call of UpdateModel.
func (mr *MockRepositoryMockRecorder) UpdateModel(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModel", reflect.TypeOf((*MockRepository)(nil).UpdateModel), arg0, arg1)
}
