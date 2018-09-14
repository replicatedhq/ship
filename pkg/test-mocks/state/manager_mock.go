// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/replicatedhq/ship/pkg/state (interfaces: Manager)

// Package state is a generated GoMock package.
package state

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "github.com/replicatedhq/ship/pkg/api"
	state "github.com/replicatedhq/ship/pkg/state"
)

// MockManager is a mock of Manager interface
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// RemoveStateFile mocks base method
func (m *MockManager) RemoveStateFile() error {
	ret := m.ctrl.Call(m, "RemoveStateFile")
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveStateFile indicates an expected call of RemoveStateFile
func (mr *MockManagerMockRecorder) RemoveStateFile() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveStateFile", reflect.TypeOf((*MockManager)(nil).RemoveStateFile))
}

// Save mocks base method
func (m *MockManager) Save(arg0 state.VersionedState) error {
	ret := m.ctrl.Call(m, "Save", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save
func (mr *MockManagerMockRecorder) Save(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockManager)(nil).Save), arg0)
}

// SaveKustomize mocks base method
func (m *MockManager) SaveKustomize(arg0 *state.Kustomize) error {
	ret := m.ctrl.Call(m, "SaveKustomize", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveKustomize indicates an expected call of SaveKustomize
func (mr *MockManagerMockRecorder) SaveKustomize(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveKustomize", reflect.TypeOf((*MockManager)(nil).SaveKustomize), arg0)
}

// SerializeConfig mocks base method
func (m *MockManager) SerializeConfig(arg0 []api.Asset, arg1 api.ReleaseMetadata, arg2 map[string]interface{}) error {
	ret := m.ctrl.Call(m, "SerializeConfig", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SerializeConfig indicates an expected call of SerializeConfig
func (mr *MockManagerMockRecorder) SerializeConfig(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SerializeConfig", reflect.TypeOf((*MockManager)(nil).SerializeConfig), arg0, arg1, arg2)
}

// SerializeContentSHA mocks base method
func (m *MockManager) SerializeContentSHA(arg0 string) error {
	ret := m.ctrl.Call(m, "SerializeContentSHA", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SerializeContentSHA indicates an expected call of SerializeContentSHA
func (mr *MockManagerMockRecorder) SerializeContentSHA(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SerializeContentSHA", reflect.TypeOf((*MockManager)(nil).SerializeContentSHA), arg0)
}

// SerializeHelmValues mocks base method
func (m *MockManager) SerializeHelmValues(arg0, arg1 string) error {
	ret := m.ctrl.Call(m, "SerializeHelmValues", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SerializeHelmValues indicates an expected call of SerializeHelmValues
func (mr *MockManagerMockRecorder) SerializeHelmValues(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SerializeHelmValues", reflect.TypeOf((*MockManager)(nil).SerializeHelmValues), arg0, arg1)
}

// SerializeMetadata mocks base method
func (m *MockManager) SerializeMetadata(arg0 *api.ShipAppMetadata) error {
	ret := m.ctrl.Call(m, "SerializeMetadata", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SerializeMetadata indicates an expected call of SerializeMetadata
func (mr *MockManagerMockRecorder) SerializeMetadata(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SerializeMetadata", reflect.TypeOf((*MockManager)(nil).SerializeMetadata), arg0)
}

// SerializeUpstream mocks base method
func (m *MockManager) SerializeUpstream(arg0 string) error {
	ret := m.ctrl.Call(m, "SerializeUpstream", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SerializeUpstream indicates an expected call of SerializeUpstream
func (mr *MockManagerMockRecorder) SerializeUpstream(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SerializeUpstream", reflect.TypeOf((*MockManager)(nil).SerializeUpstream), arg0)
}

// TryLoad mocks base method
func (m *MockManager) TryLoad() (state.State, error) {
	ret := m.ctrl.Call(m, "TryLoad")
	ret0, _ := ret[0].(state.State)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TryLoad indicates an expected call of TryLoad
func (mr *MockManagerMockRecorder) TryLoad() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TryLoad", reflect.TypeOf((*MockManager)(nil).TryLoad))
}
