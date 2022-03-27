// Code generated by MockGen. DO NOT EDIT.
// Source: ./pkg/services/model.go

// Package rpc is a generated GoMock package.
package handler

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	"github.com/instill-ai/model-backend/pkg/datamodel"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

// MockModelService is a mock of ModelService interface.
type MockModelService struct {
	ctrl     *gomock.Controller
	recorder *MockModelServiceMockRecorder
}

// MockModelServiceMockRecorder is the mock recorder for MockModelService.
type MockModelServiceMockRecorder struct {
	mock *MockModelService
}

// NewMockModelService creates a new mock instance.
func NewMockModelService(ctrl *gomock.Controller) *MockModelService {
	mock := &MockModelService{ctrl: ctrl}
	mock.recorder = &MockModelServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelService) EXPECT() *MockModelServiceMockRecorder {
	return m.recorder
}

// CreateModel mocks base method.
func (m *MockModelService) CreateModel(model *datamodel.Model) (*datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModel", model)
	ret0, _ := ret[0].(*datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateModel indicates an expected call of CreateModel.
func (mr *MockModelServiceMockRecorder) CreateModel(model interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModel", reflect.TypeOf((*MockModelService)(nil).CreateModel), model)
}

// CreateModelBinaryFileUpload mocks base method.
func (m *MockModelService) CreateModelBinaryFileUpload(namespace string, createdModel *datamodel.Model) (*modelPB.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateModelBinaryFileUpload", namespace, createdModel)
	ret0, _ := ret[0].(*modelPB.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateModelBinaryFileUpload indicates an expected call of CreateModelBinaryFileUpload.
func (mr *MockModelServiceMockRecorder) CreateModelBinaryFileUpload(namespace, createdModel interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateModelBinaryFileUpload", reflect.TypeOf((*MockModelService)(nil).CreateModelBinaryFileUpload), namespace, createdModel)
}

// CreateVersion mocks base method.
func (m *MockModelService) CreateVersion(version datamodel.Version) (datamodel.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVersion", version)
	ret0, _ := ret[0].(datamodel.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVersion indicates an expected call of CreateVersion.
func (mr *MockModelServiceMockRecorder) CreateVersion(version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVersion", reflect.TypeOf((*MockModelService)(nil).CreateVersion), version)
}

// DeleteModel mocks base method.
func (m *MockModelService) DeleteModel(namespace, modelName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModel", namespace, modelName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModel indicates an expected call of DeleteModel.
func (mr *MockModelServiceMockRecorder) DeleteModel(namespace, modelName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModel", reflect.TypeOf((*MockModelService)(nil).DeleteModel), namespace, modelName)
}

// DeleteModelVersion mocks base method.
func (m *MockModelService) DeleteModelVersion(namespace, modelName string, version uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteModelVersion", namespace, modelName, version)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteModelVersion indicates an expected call of DeleteModelVersion.
func (mr *MockModelServiceMockRecorder) DeleteModelVersion(namespace, modelName, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteModelVersion", reflect.TypeOf((*MockModelService)(nil).DeleteModelVersion), namespace, modelName, version)
}

// GetFullModelData mocks base method.
func (m *MockModelService) GetFullModelData(namespace, modelName string) (*modelPB.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFullModelData", namespace, modelName)
	ret0, _ := ret[0].(*modelPB.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFullModelData indicates an expected call of GetFullModelData.
func (mr *MockModelServiceMockRecorder) GetFullModelData(namespace, modelName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFullModelData", reflect.TypeOf((*MockModelService)(nil).GetFullModelData), namespace, modelName)
}

// GetModelByName mocks base method.
func (m *MockModelService) GetModelByName(namespace, modelName string) (datamodel.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelByName", namespace, modelName)
	ret0, _ := ret[0].(datamodel.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelByName indicates an expected call of GetModelByName.
func (mr *MockModelServiceMockRecorder) GetModelByName(namespace, modelName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelByName", reflect.TypeOf((*MockModelService)(nil).GetModelByName), namespace, modelName)
}

// GetModelVersion mocks base method.
func (m *MockModelService) GetModelVersion(modelId, version uint64) (datamodel.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelVersion", modelId, version)
	ret0, _ := ret[0].(datamodel.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelVersion indicates an expected call of GetModelVersion.
func (mr *MockModelServiceMockRecorder) GetModelVersion(modelId, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelVersion", reflect.TypeOf((*MockModelService)(nil).GetModelVersion), modelId, version)
}

// GetModelVersionLatest mocks base method.
func (m *MockModelService) GetModelVersionLatest(modelId uint64) (datamodel.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelVersionLatest", modelId)
	ret0, _ := ret[0].(datamodel.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelVersionLatest indicates an expected call of GetModelVersionLatest.
func (mr *MockModelServiceMockRecorder) GetModelVersionLatest(modelId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelVersionLatest", reflect.TypeOf((*MockModelService)(nil).GetModelVersionLatest), modelId)
}

// GetModelVersions mocks base method.
func (m *MockModelService) GetModelVersions(modelId uint64) ([]datamodel.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelVersions", modelId)
	ret0, _ := ret[0].([]datamodel.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelVersions indicates an expected call of GetModelVersions.
func (mr *MockModelServiceMockRecorder) GetModelVersions(modelId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelVersions", reflect.TypeOf((*MockModelService)(nil).GetModelVersions), modelId)
}

// ListModels mocks base method.
func (m *MockModelService) ListModels(namespace string) ([]*modelPB.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListModels", namespace)
	ret0, _ := ret[0].([]*modelPB.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListModels indicates an expected call of ListModels.
func (mr *MockModelServiceMockRecorder) ListModels(namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListModels", reflect.TypeOf((*MockModelService)(nil).ListModels), namespace)
}

// ModelInfer mocks base method.
func (m *MockModelService) ModelInfer(namespace, modelName string, version uint64, imgsBytes [][]byte, task modelPB.Model_Task) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelInfer", namespace, modelName, version, imgsBytes, task)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelInfer indicates an expected call of ModelInfer.
func (mr *MockModelServiceMockRecorder) ModelInfer(namespace, modelName, version, imgsBytes, task interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelInfer", reflect.TypeOf((*MockModelService)(nil).ModelInfer), namespace, modelName, version, imgsBytes, task)
}

// UpdateModelVersion mocks base method.
func (m *MockModelService) UpdateModelVersion(namespace string, updatedInfo *modelPB.UpdateModelVersionRequest) (*modelPB.ModelVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateModelVersion", namespace, updatedInfo)
	ret0, _ := ret[0].(*modelPB.ModelVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateModelVersion indicates an expected call of UpdateModelVersion.
func (mr *MockModelServiceMockRecorder) UpdateModelVersion(namespace, updatedInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelVersion", reflect.TypeOf((*MockModelService)(nil).UpdateModelVersion), namespace, updatedInfo)
}
