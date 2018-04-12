// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/lifecycle/render/plan/planner.go

// Package planner is a generated GoMock package.
package planner

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "github.com/replicatedcom/ship/pkg/api"
	plan "github.com/replicatedcom/ship/pkg/lifecycle/render/plan"
)

// MockPlanner is a mock of Planner interface
type MockPlanner struct {
	ctrl     *gomock.Controller
	recorder *MockPlannerMockRecorder
}

// MockPlannerMockRecorder is the mock recorder for MockPlanner
type MockPlannerMockRecorder struct {
	mock *MockPlanner
}

// NewMockPlanner creates a new mock instance
func NewMockPlanner(ctrl *gomock.Controller) *MockPlanner {
	mock := &MockPlanner{ctrl: ctrl}
	mock.recorder = &MockPlannerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPlanner) EXPECT() *MockPlannerMockRecorder {
	return m.recorder
}

// Build mocks base method
func (m *MockPlanner) Build(assets []api.Asset, config map[string]interface{}) plan.Plan {
	ret := m.ctrl.Call(m, "Build", assets, config)
	ret0, _ := ret[0].(plan.Plan)
	return ret0
}

// Build indicates an expected call of Build
func (mr *MockPlannerMockRecorder) Build(assets, config interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Build", reflect.TypeOf((*MockPlanner)(nil).Build), assets, config)
}

// Confirm mocks base method
func (m *MockPlanner) Confirm(arg0 plan.Plan) (bool, error) {
	ret := m.ctrl.Call(m, "Confirm", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Confirm indicates an expected call of Confirm
func (mr *MockPlannerMockRecorder) Confirm(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Confirm", reflect.TypeOf((*MockPlanner)(nil).Confirm), arg0)
}

// Execute mocks base method
func (m *MockPlanner) Execute(arg0 context.Context, arg1 plan.Plan) error {
	ret := m.ctrl.Call(m, "Execute", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Execute indicates an expected call of Execute
func (mr *MockPlannerMockRecorder) Execute(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockPlanner)(nil).Execute), arg0, arg1)
}
