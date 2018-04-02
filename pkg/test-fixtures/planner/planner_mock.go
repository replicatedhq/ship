package planner

/*
DO NOT EDIT!
This code was generated automatically using github.com/gojuno/minimock v1.9
The original interface "IPlanner" can be found in github.com/replicatedcom/ship/pkg/lifecycle/render/plan
*/
import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gojuno/minimock"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/plan"
	testify_assert "github.com/stretchr/testify/assert"
)

//IPlannerMock implements github.com/replicatedcom/ship/pkg/lifecycle/render/plan.IPlanner
type IPlannerMock struct {
	t minimock.Tester

	BuildFunc       func(p []api.Asset, p1 map[string]interface{}) (r plan.Plan)
	BuildCounter    uint64
	BuildPreCounter uint64
	BuildMock       mIPlannerMockBuild

	ConfirmFunc       func(p plan.Plan) (r bool, r1 error)
	ConfirmCounter    uint64
	ConfirmPreCounter uint64
	ConfirmMock       mIPlannerMockConfirm

	ExecuteFunc       func(p context.Context, p1 plan.Plan) (r error)
	ExecuteCounter    uint64
	ExecutePreCounter uint64
	ExecuteMock       mIPlannerMockExecute
}

//NewIPlannerMock returns a mock for github.com/replicatedcom/ship/pkg/lifecycle/render/plan.IPlanner
func NewIPlannerMock(t minimock.Tester) *IPlannerMock {
	m := &IPlannerMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.BuildMock = mIPlannerMockBuild{mock: m}
	m.ConfirmMock = mIPlannerMockConfirm{mock: m}
	m.ExecuteMock = mIPlannerMockExecute{mock: m}

	return m
}

type mIPlannerMockBuild struct {
	mock             *IPlannerMock
	mockExpectations *IPlannerMockBuildParams
}

//IPlannerMockBuildParams represents input parameters of the IPlanner.Build
type IPlannerMockBuildParams struct {
	p  []api.Asset
	p1 map[string]interface{}
}

//Expect sets up expected params for the IPlanner.Build
func (m *mIPlannerMockBuild) Expect(p []api.Asset, p1 map[string]interface{}) *mIPlannerMockBuild {
	m.mockExpectations = &IPlannerMockBuildParams{p, p1}
	return m
}

//Return sets up a mock for IPlanner.Build to return Return's arguments
func (m *mIPlannerMockBuild) Return(r plan.Plan) *IPlannerMock {
	m.mock.BuildFunc = func(p []api.Asset, p1 map[string]interface{}) plan.Plan {
		return r
	}
	return m.mock
}

//Set uses given function f as a mock of IPlanner.Build method
func (m *mIPlannerMockBuild) Set(f func(p []api.Asset, p1 map[string]interface{}) (r plan.Plan)) *IPlannerMock {
	m.mock.BuildFunc = f
	return m.mock
}

//Build implements github.com/replicatedcom/ship/pkg/lifecycle/render/plan.IPlanner interface
func (m *IPlannerMock) Build(p []api.Asset, p1 map[string]interface{}) (r plan.Plan) {
	atomic.AddUint64(&m.BuildPreCounter, 1)
	defer atomic.AddUint64(&m.BuildCounter, 1)

	if m.BuildMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.BuildMock.mockExpectations, IPlannerMockBuildParams{p, p1},
			"IPlanner.Build got unexpected parameters")

		if m.BuildFunc == nil {

			m.t.Fatal("No results are set for the IPlannerMock.Build")

			return
		}
	}

	if m.BuildFunc == nil {
		m.t.Fatal("Unexpected call to IPlannerMock.Build")
		return
	}

	return m.BuildFunc(p, p1)
}

//BuildMinimockCounter returns a count of IPlannerMock.BuildFunc invocations
func (m *IPlannerMock) BuildMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.BuildCounter)
}

//BuildMinimockPreCounter returns the value of IPlannerMock.Build invocations
func (m *IPlannerMock) BuildMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.BuildPreCounter)
}

type mIPlannerMockConfirm struct {
	mock             *IPlannerMock
	mockExpectations *IPlannerMockConfirmParams
}

//IPlannerMockConfirmParams represents input parameters of the IPlanner.Confirm
type IPlannerMockConfirmParams struct {
	p plan.Plan
}

//Expect sets up expected params for the IPlanner.Confirm
func (m *mIPlannerMockConfirm) Expect(p plan.Plan) *mIPlannerMockConfirm {
	m.mockExpectations = &IPlannerMockConfirmParams{p}
	return m
}

//Return sets up a mock for IPlanner.Confirm to return Return's arguments
func (m *mIPlannerMockConfirm) Return(r bool, r1 error) *IPlannerMock {
	m.mock.ConfirmFunc = func(p plan.Plan) (bool, error) {
		return r, r1
	}
	return m.mock
}

//Set uses given function f as a mock of IPlanner.Confirm method
func (m *mIPlannerMockConfirm) Set(f func(p plan.Plan) (r bool, r1 error)) *IPlannerMock {
	m.mock.ConfirmFunc = f
	return m.mock
}

//Confirm implements github.com/replicatedcom/ship/pkg/lifecycle/render/plan.IPlanner interface
func (m *IPlannerMock) Confirm(p plan.Plan) (r bool, r1 error) {
	atomic.AddUint64(&m.ConfirmPreCounter, 1)
	defer atomic.AddUint64(&m.ConfirmCounter, 1)

	if m.ConfirmMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.ConfirmMock.mockExpectations, IPlannerMockConfirmParams{p},
			"IPlanner.Confirm got unexpected parameters")

		if m.ConfirmFunc == nil {

			m.t.Fatal("No results are set for the IPlannerMock.Confirm")

			return
		}
	}

	if m.ConfirmFunc == nil {
		m.t.Fatal("Unexpected call to IPlannerMock.Confirm")
		return
	}

	return m.ConfirmFunc(p)
}

//ConfirmMinimockCounter returns a count of IPlannerMock.ConfirmFunc invocations
func (m *IPlannerMock) ConfirmMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.ConfirmCounter)
}

//ConfirmMinimockPreCounter returns the value of IPlannerMock.Confirm invocations
func (m *IPlannerMock) ConfirmMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.ConfirmPreCounter)
}

type mIPlannerMockExecute struct {
	mock             *IPlannerMock
	mockExpectations *IPlannerMockExecuteParams
}

//IPlannerMockExecuteParams represents input parameters of the IPlanner.Execute
type IPlannerMockExecuteParams struct {
	p  context.Context
	p1 plan.Plan
}

//Expect sets up expected params for the IPlanner.Execute
func (m *mIPlannerMockExecute) Expect(p context.Context, p1 plan.Plan) *mIPlannerMockExecute {
	m.mockExpectations = &IPlannerMockExecuteParams{p, p1}
	return m
}

//Return sets up a mock for IPlanner.Execute to return Return's arguments
func (m *mIPlannerMockExecute) Return(r error) *IPlannerMock {
	m.mock.ExecuteFunc = func(p context.Context, p1 plan.Plan) error {
		return r
	}
	return m.mock
}

//Set uses given function f as a mock of IPlanner.Execute method
func (m *mIPlannerMockExecute) Set(f func(p context.Context, p1 plan.Plan) (r error)) *IPlannerMock {
	m.mock.ExecuteFunc = f
	return m.mock
}

//Execute implements github.com/replicatedcom/ship/pkg/lifecycle/render/plan.IPlanner interface
func (m *IPlannerMock) Execute(p context.Context, p1 plan.Plan) (r error) {
	atomic.AddUint64(&m.ExecutePreCounter, 1)
	defer atomic.AddUint64(&m.ExecuteCounter, 1)

	if m.ExecuteMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.ExecuteMock.mockExpectations, IPlannerMockExecuteParams{p, p1},
			"IPlanner.Execute got unexpected parameters")

		if m.ExecuteFunc == nil {

			m.t.Fatal("No results are set for the IPlannerMock.Execute")

			return
		}
	}

	if m.ExecuteFunc == nil {
		m.t.Fatal("Unexpected call to IPlannerMock.Execute")
		return
	}

	return m.ExecuteFunc(p, p1)
}

//ExecuteMinimockCounter returns a count of IPlannerMock.ExecuteFunc invocations
func (m *IPlannerMock) ExecuteMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.ExecuteCounter)
}

//ExecuteMinimockPreCounter returns the value of IPlannerMock.Execute invocations
func (m *IPlannerMock) ExecuteMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.ExecutePreCounter)
}

//ValidateCallCounters checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *IPlannerMock) ValidateCallCounters() {

	if m.BuildFunc != nil && atomic.LoadUint64(&m.BuildCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Build")
	}

	if m.ConfirmFunc != nil && atomic.LoadUint64(&m.ConfirmCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Confirm")
	}

	if m.ExecuteFunc != nil && atomic.LoadUint64(&m.ExecuteCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Execute")
	}

}

//CheckMocksCalled checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *IPlannerMock) CheckMocksCalled() {
	m.Finish()
}

//Finish checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish or use Finish method of minimock.Controller
func (m *IPlannerMock) Finish() {
	m.MinimockFinish()
}

//MinimockFinish checks that all mocked methods of the interface have been called at least once
func (m *IPlannerMock) MinimockFinish() {

	if m.BuildFunc != nil && atomic.LoadUint64(&m.BuildCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Build")
	}

	if m.ConfirmFunc != nil && atomic.LoadUint64(&m.ConfirmCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Confirm")
	}

	if m.ExecuteFunc != nil && atomic.LoadUint64(&m.ExecuteCounter) == 0 {
		m.t.Fatal("Expected call to IPlannerMock.Execute")
	}

}

//Wait waits for all mocked methods to be called at least once
//Deprecated: please use MinimockWait or use Wait method of minimock.Controller
func (m *IPlannerMock) Wait(timeout time.Duration) {
	m.MinimockWait(timeout)
}

//MinimockWait waits for all mocked methods to be called at least once
//this method is called by minimock.Controller
func (m *IPlannerMock) MinimockWait(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		ok := true
		ok = ok && (m.BuildFunc == nil || atomic.LoadUint64(&m.BuildCounter) > 0)
		ok = ok && (m.ConfirmFunc == nil || atomic.LoadUint64(&m.ConfirmCounter) > 0)
		ok = ok && (m.ExecuteFunc == nil || atomic.LoadUint64(&m.ExecuteCounter) > 0)

		if ok {
			return
		}

		select {
		case <-timeoutCh:

			if m.BuildFunc != nil && atomic.LoadUint64(&m.BuildCounter) == 0 {
				m.t.Error("Expected call to IPlannerMock.Build")
			}

			if m.ConfirmFunc != nil && atomic.LoadUint64(&m.ConfirmCounter) == 0 {
				m.t.Error("Expected call to IPlannerMock.Confirm")
			}

			if m.ExecuteFunc != nil && atomic.LoadUint64(&m.ExecuteCounter) == 0 {
				m.t.Error("Expected call to IPlannerMock.Execute")
			}

			m.t.Fatalf("Some mocks were not called on time: %s", timeout)
			return
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

//AllMocksCalled returns true if all mocked methods were called before the execution of AllMocksCalled,
//it can be used with assert/require, i.e. assert.True(mock.AllMocksCalled())
func (m *IPlannerMock) AllMocksCalled() bool {

	if m.BuildFunc != nil && atomic.LoadUint64(&m.BuildCounter) == 0 {
		return false
	}

	if m.ConfirmFunc != nil && atomic.LoadUint64(&m.ConfirmCounter) == 0 {
		return false
	}

	if m.ExecuteFunc != nil && atomic.LoadUint64(&m.ExecuteCounter) == 0 {
		return false
	}

	return true
}
