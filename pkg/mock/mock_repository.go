// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/model-backend/pkg/repository (interfaces: Repository)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	datamodel "github.com/instill-ai/model-backend/pkg/datamodel"
	modelv1alpha "github.com/instill-ai/protogen-go/model/model/v1alpha"
	filtering "go.einride.tech/aip/filtering"
	ordering "go.einride.tech/aip/ordering"
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

// CreateModelTags mocks base method.
func (m *MockRepository) CreateModelTags(arg0 context.Context, arg1 uuid.UUID, arg2 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelTags", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateModelTags indicates an expected call of CreateModelTags.
func (mr *MockRepositoryMockRecorder) CreateModelTags(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelTags", reflect.TypeOf((*MockRepository)(nil).CreateModelTags), arg0, arg1, arg2)
}

// CreateModelTrigger mocks base method.
func (m *MockRepository) CreateModelTrigger(arg0 context.Context, arg1 *datamodel.ModelTrigger) (*datamodel.ModelTrigger, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelTrigger", arg0, arg1)
	ret0, _ := ret[0].(*datamodel.ModelTrigger)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateModelTrigger indicates an expected call of CreateModelTrigger.
func (mr *MockRepositoryMockRecorder) CreateModelTrigger(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelTrigger", reflect.TypeOf((*MockRepository)(nil).CreateModelTrigger), arg0, arg1)
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

// DeleteModelTags mocks base method.
func (m *MockRepository) DeleteModelTags(arg0 context.Context, arg1 uuid.UUID, arg2 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModelTags", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModelTags indicates an expected call of DeleteModelTags.
func (mr *MockRepositoryMockRecorder) DeleteModelTags(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModelTags", reflect.TypeOf((*MockRepository)(nil).DeleteModelTags), arg0, arg1, arg2)
}

// DeleteModelVersionByDigest mocks base method.
func (m *MockRepository) DeleteModelVersionByDigest(arg0 context.Context, arg1 uuid.UUID, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModelVersionByDigest", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModelVersionByDigest indicates an expected call of DeleteModelVersionByDigest.
func (mr *MockRepositoryMockRecorder) DeleteModelVersionByDigest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModelVersionByDigest", reflect.TypeOf((*MockRepository)(nil).DeleteModelVersionByDigest), arg0, arg1, arg2)
}

// DeleteModelVersionByID mocks base method.
func (m *MockRepository) DeleteModelVersionByID(arg0 context.Context, arg1 uuid.UUID, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModelVersionByID", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModelVersionByID indicates an expected call of DeleteModelVersionByID.
func (mr *MockRepositoryMockRecorder) DeleteModelVersionByID(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModelVersionByID", reflect.TypeOf((*MockRepository)(nil).DeleteModelVersionByID), arg0, arg1, arg2)
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

// GetLatestModelVersionByModelUID mocks base method.
func (m *MockRepository) GetLatestModelVersionByModelUID(arg0 context.Context, arg1 uuid.UUID) (*datamodel.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestModelVersionByModelUID", arg0, arg1)
	ret0, _ := ret[0].(*datamodel.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestModelVersionByModelUID indicates an expected call of GetLatestModelVersionByModelUID.
func (mr *MockRepositoryMockRecorder) GetLatestModelVersionByModelUID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestModelVersionByModelUID", reflect.TypeOf((*MockRepository)(nil).GetLatestModelVersionByModelUID), arg0, arg1)
}

// GetModelByUID mocks base method.
func (m *MockRepository) GetModelByUID(arg0 context.Context, arg1 uuid.UUID, arg2, arg3 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUID", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUID indicates an expected call of GetModelByUID.
func (mr *MockRepositoryMockRecorder) GetModelByUID(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUID", reflect.TypeOf((*MockRepository)(nil).GetModelByUID), arg0, arg1, arg2, arg3)
}

// GetModelByUIDAdmin mocks base method.
func (m *MockRepository) GetModelByUIDAdmin(arg0 context.Context, arg1 uuid.UUID, arg2, arg3 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByUIDAdmin", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByUIDAdmin indicates an expected call of GetModelByUIDAdmin.
func (mr *MockRepositoryMockRecorder) GetModelByUIDAdmin(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByUIDAdmin", reflect.TypeOf((*MockRepository)(nil).GetModelByUIDAdmin), arg0, arg1, arg2, arg3)
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
func (m *MockRepository) GetNamespaceModelByID(arg0 context.Context, arg1, arg2 string, arg3, arg4 bool) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNamespaceModelByID", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNamespaceModelByID indicates an expected call of GetNamespaceModelByID.
func (mr *MockRepositoryMockRecorder) GetNamespaceModelByID(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNamespaceModelByID", reflect.TypeOf((*MockRepository)(nil).GetNamespaceModelByID), arg0, arg1, arg2, arg3, arg4)
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

// ListModelTags mocks base method.
func (m *MockRepository) ListModelTags(arg0 context.Context, arg1 uuid.UUID) ([]datamodel.ModelTag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelTags", arg0, arg1)
	ret0, _ := ret[0].([]datamodel.ModelTag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModelTags indicates an expected call of ListModelTags.
func (mr *MockRepositoryMockRecorder) ListModelTags(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelTags", reflect.TypeOf((*MockRepository)(nil).ListModelTags), arg0, arg1)
}

// ListModelTriggers mocks base method.
func (m *MockRepository) ListModelTriggers(arg0 context.Context, arg1, arg2 int64, arg3 filtering.Filter, arg4 ordering.OrderBy, arg5 string, arg6 bool, arg7 string) ([]*datamodel.ModelTrigger, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelTriggers", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
	ret0, _ := ret[0].([]*datamodel.ModelTrigger)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ListModelTriggers indicates an expected call of ListModelTriggers.
func (mr *MockRepositoryMockRecorder) ListModelTriggers(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelTriggers", reflect.TypeOf((*MockRepository)(nil).ListModelTriggers), arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// ListModelVersions mocks base method.
func (m *MockRepository) ListModelVersions(arg0 context.Context, arg1 uuid.UUID, arg2 bool) ([]*datamodel.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelVersions", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*datamodel.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModelVersions indicates an expected call of ListModelVersions.
func (mr *MockRepositoryMockRecorder) ListModelVersions(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelVersions", reflect.TypeOf((*MockRepository)(nil).ListModelVersions), arg0, arg1, arg2)
}

// ListModelVersionsByDigest mocks base method.
func (m *MockRepository) ListModelVersionsByDigest(arg0 context.Context, arg1 uuid.UUID, arg2 string) ([]*datamodel.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelVersionsByDigest", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*datamodel.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModelVersionsByDigest indicates an expected call of ListModelVersionsByDigest.
func (mr *MockRepositoryMockRecorder) ListModelVersionsByDigest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelVersionsByDigest", reflect.TypeOf((*MockRepository)(nil).ListModelVersionsByDigest), arg0, arg1, arg2)
}

// ListModels mocks base method.
func (m *MockRepository) ListModels(arg0 context.Context, arg1 int64, arg2 string, arg3 bool, arg4 filtering.Filter, arg5 []uuid.UUID, arg6 bool, arg7 ordering.OrderBy, arg8 *modelv1alpha.Model_Visibility) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModels", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModels indicates an expected call of ListModels.
func (mr *MockRepositoryMockRecorder) ListModels(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModels", reflect.TypeOf((*MockRepository)(nil).ListModels), arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
}

// ListModelsAdmin mocks base method.
func (m *MockRepository) ListModelsAdmin(arg0 context.Context, arg1 int64, arg2 string, arg3 bool, arg4 filtering.Filter, arg5 bool) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModelsAdmin", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListModelsAdmin indicates an expected call of ListModelsAdmin.
func (mr *MockRepositoryMockRecorder) ListModelsAdmin(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModelsAdmin", reflect.TypeOf((*MockRepository)(nil).ListModelsAdmin), arg0, arg1, arg2, arg3, arg4, arg5)
}

// ListNamespaceModels mocks base method.
func (m *MockRepository) ListNamespaceModels(arg0 context.Context, arg1 string, arg2 int64, arg3 string, arg4 bool, arg5 filtering.Filter, arg6 []uuid.UUID, arg7 bool, arg8 ordering.OrderBy, arg9 *modelv1alpha.Model_Visibility) ([]*datamodel.Model, int64, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNamespaceModels", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9)
	ret0, _ := ret[0].([]*datamodel.Model)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ListNamespaceModels indicates an expected call of ListNamespaceModels.
func (mr *MockRepositoryMockRecorder) ListNamespaceModels(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNamespaceModels", reflect.TypeOf((*MockRepository)(nil).ListNamespaceModels), arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9)
}

// UpdateModelTrigger mocks base method.
func (m *MockRepository) UpdateModelTrigger(arg0 context.Context, arg1 *datamodel.ModelTrigger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModelTrigger", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModelTrigger indicates an expected call of UpdateModelTrigger.
func (mr *MockRepositoryMockRecorder) UpdateModelTrigger(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelTrigger", reflect.TypeOf((*MockRepository)(nil).UpdateModelTrigger), arg0, arg1)
}

// UpdateModelVersionDigestByID mocks base method.
func (m *MockRepository) UpdateModelVersionDigestByID(arg0 context.Context, arg1 uuid.UUID, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModelVersionDigestByID", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModelVersionDigestByID indicates an expected call of UpdateModelVersionDigestByID.
func (mr *MockRepositoryMockRecorder) UpdateModelVersionDigestByID(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelVersionDigestByID", reflect.TypeOf((*MockRepository)(nil).UpdateModelVersionDigestByID), arg0, arg1, arg2, arg3)
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
