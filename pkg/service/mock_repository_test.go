// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/pkg/repository (interfaces: Repository)

// Package service_test is a generated GoMock package.
package service_test

import (
	context "context"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	modelv1alpha "github.com/instill-ai/protogen-go/model/model/v1alpha"
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

// CreateModelPrediction mocks base method.
func (m *MockRepository) CreateModelPrediction(arg0 context.Context, arg1 *datamodel.ModelPrediction) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelPrediction", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateModelPrediction indicates an expected call of CreateModelPrediction.
func (mr *MockRepositoryMockRecorder) CreateModelPrediction(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelPrediction", reflect.TypeOf((*MockRepository)(nil).CreateModelPrediction), arg0, arg1)
}

// CreateModelVersion mocks base method.
func (m *MockRepository) CreateModelVersion(arg0 context.Context, arg1 string, arg2 *datamodel.ModelVersion) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelVersion", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateModelVersion indicates an expected call of CreateModelVersion.
func (mr *MockRepositoryMockRecorder) CreateModelVersion(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelVersion", reflect.TypeOf((*MockRepository)(nil).CreateModelVersion), arg0, arg1, arg2)
}

// CreateNamespaceModel mocks base method.
func (m *MockRepository) CreateNamespaceModel(arg0 context.Context, arg1 string, arg2 *datamodel.Model) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNamespaceModel", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateNamespaceModel indicates an expected call of CreateNamespaceModel.
func (mr *MockRepositoryMockRecorder) CreateNamespaceModel(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNamespaceModel", reflect.TypeOf((*MockRepository)(nil).CreateNamespaceModel), arg0, arg1, arg2)
}

// DeleteModelVersion mocks base method.
func (m *MockRepository) DeleteModelVersion(arg0 context.Context, arg1 string, arg2 *datamodel.ModelVersion) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModelVersion", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModelVersion indicates an expected call of DeleteModelVersion.
func (mr *MockRepositoryMockRecorder) DeleteModelVersion(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModelVersion", reflect.TypeOf((*MockRepository)(nil).DeleteModelVersion), arg0, arg1, arg2)
}

// DeleteNamespaceModelByID mocks base method.
func (m *MockRepository) DeleteNamespaceModelByID(arg0 context.Context, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNamespaceModelByID", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNamespaceModelByID indicates an expected call of DeleteNamespaceModelByID.
func (mr *MockRepositoryMockRecorder) DeleteNamespaceModelByID(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNamespaceModelByID", reflect.TypeOf((*MockRepository)(nil).DeleteNamespaceModelByID), arg0, arg1, arg2)
}

// GetModelByIDAdmin mocks base method.
func (m *MockRepository) GetModelByIDAdmin(arg0 context.Context, arg1 string, arg2 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByIDAdmin", arg0, arg1, arg2)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByIDAdmin indicates an expected call of GetModelByIDAdmin.
func (mr *MockRepositoryMockRecorder) GetModelByIDAdmin(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByIDAdmin", reflect.TypeOf((*MockRepository)(nil).GetModelByIDAdmin), arg0, arg1, arg2)
}

// GetModelByUID mocks base method.
func (m *MockRepository) GetModelByUID(arg0 context.Context, arg1 uuid.UUID, arg2 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUID", arg0, arg1, arg2)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUID indicates an expected call of GetModelByUID.
func (mr *MockRepositoryMockRecorder) GetModelByUID(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUID", reflect.TypeOf((*MockRepository)(nil).GetModelByUID), arg0, arg1, arg2)
}

// GetModelByUIDAdmin mocks base method.
func (m *MockRepository) GetModelByUIDAdmin(arg0 context.Context, arg1 uuid.UUID, arg2 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUIDAdmin", arg0, arg1, arg2)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUIDAdmin indicates an expected call of GetModelByUIDAdmin.
func (mr *MockRepositoryMockRecorder) GetModelByUIDAdmin(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUIDAdmin", reflect.TypeOf((*MockRepository)(nil).GetModelByUIDAdmin), arg0, arg1, arg2)
}

// GetModelDefinition mocks base method.
func (m *MockRepository) GetModelDefinition(arg0 string) (*datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinition", arg0)
	ret0, _ := ret[0].(*datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinition indicates an expected call of GetModelDefinition.
func (mr *MockRepositoryMockRecorder) GetModelDefinition(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinition", reflect.TypeOf((*MockRepository)(nil).GetModelDefinition), arg0)
}

// GetModelDefinitionByUID mocks base method.
func (m *MockRepository) GetModelDefinitionByUID(arg0 uuid.UUID) (*datamodel.ModelDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelDefinitionByUID", arg0)
	ret0, _ := ret[0].(*datamodel.ModelDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelDefinitionByUID indicates an expected call of GetModelDefinitionByUID.
func (mr *MockRepositoryMockRecorder) GetModelDefinitionByUID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelDefinitionByUID", reflect.TypeOf((*MockRepository)(nil).GetModelDefinitionByUID), arg0)
}

// GetModelVersionByID mocks base method.
func (m *MockRepository) GetModelVersionByID(arg0 context.Context, arg1 uuid.UUID, arg2 string) (*datamodel.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelVersionByID", arg0, arg1, arg2)
	ret0, _ := ret[0].(*datamodel.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelVersionByID indicates an expected call of GetModelVersionByID.
func (mr *MockRepositoryMockRecorder) GetModelVersionByID(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelVersionByID", reflect.TypeOf((*MockRepository)(nil).GetModelVersionByID), arg0, arg1, arg2)
}

// GetNamespaceModelByID mocks base method.
func (m *MockRepository) GetNamespaceModelByID(arg0 context.Context, arg1, arg2 string, arg3 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNamespaceModelByID", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNamespaceModelByID indicates an expected call of GetNamespaceModelByID.
func (mr *MockRepositoryMockRecorder) GetNamespaceModelByID(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNamespaceModelByID", reflect.TypeOf((*MockRepository)(nil).GetNamespaceModelByID), arg0, arg1, arg2, arg3)
}

// ListModelDefinitions mocks base method.
func (m *MockRepository) ListModelDefinitions(arg0 modelv1alpha.View, arg1 int64, arg2 string) ([]*datamodel.ModelDefinition, string, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelDefinitions", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*datamodel.ModelDefinition)
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

// ListModelVerions mocks base method.
func (m *MockRepository) ListModelVerions(arg0 context.Context, arg1 uuid.UUID) ([]*datamodel.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelVerions", arg0, arg1)
	ret0, _ := ret[0].([]*datamodel.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModelVerions indicates an expected call of ListModelVerions.
func (mr *MockRepositoryMockRecorder) ListModelVerions(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelVerions", reflect.TypeOf((*MockRepository)(nil).ListModelVerions), arg0, arg1)
}

// ListModels mocks base method.
func (m *MockRepository) ListModels(arg0 context.Context, arg1 int64, arg2 string, arg3 bool, arg4 []uuid.UUID, arg5 bool) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModels", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModels indicates an expected call of ListModels.
func (mr *MockRepositoryMockRecorder) ListModels(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModels", reflect.TypeOf((*MockRepository)(nil).ListModels), arg0, arg1, arg2, arg3, arg4, arg5)
}

// ListModelsAdmin mocks base method.
func (m *MockRepository) ListModelsAdmin(arg0 context.Context, arg1 int64, arg2 string, arg3, arg4 bool) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelsAdmin", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelsAdmin indicates an expected call of ListModelsAdmin.
func (mr *MockRepositoryMockRecorder) ListModelsAdmin(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelsAdmin", reflect.TypeOf((*MockRepository)(nil).ListModelsAdmin), arg0, arg1, arg2, arg3, arg4)
}

// ListNamespaceModels mocks base method.
func (m *MockRepository) ListNamespaceModels(arg0 context.Context, arg1 string, arg2 int64, arg3 string, arg4 bool, arg5 []uuid.UUID, arg6 bool) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNamespaceModels", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListNamespaceModels indicates an expected call of ListNamespaceModels.
func (mr *MockRepositoryMockRecorder) ListNamespaceModels(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNamespaceModels", reflect.TypeOf((*MockRepository)(nil).ListNamespaceModels), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// UpdateNamespaceModelByID mocks base method.
func (m *MockRepository) UpdateNamespaceModelByID(arg0 context.Context, arg1, arg2 string, arg3 *datamodel.Model) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNamespaceModelByID", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateNamespaceModelByID indicates an expected call of UpdateNamespaceModelByID.
func (mr *MockRepositoryMockRecorder) UpdateNamespaceModelByID(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNamespaceModelByID", reflect.TypeOf((*MockRepository)(nil).UpdateNamespaceModelByID), arg0, arg1, arg2, arg3)
}

// UpdateNamespaceModelIDByID mocks base method.
func (m *MockRepository) UpdateNamespaceModelIDByID(arg0 context.Context, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNamespaceModelIDByID", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateNamespaceModelIDByID indicates an expected call of UpdateNamespaceModelIDByID.
func (mr *MockRepositoryMockRecorder) UpdateNamespaceModelIDByID(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNamespaceModelIDByID", reflect.TypeOf((*MockRepository)(nil).UpdateNamespaceModelIDByID), arg0, arg1, arg2, arg3)
}
