package config

/*
DO NOT EDIT!
This code was generated automatically using github.com/gojuno/minimock v1.9
The original interface "IResolver" can be found in github.com/replicatedcom/ship/pkg/lifecycle/render/config
*/

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gojuno/minimock"
	testify_assert "github.com/stretchr/testify/assert"
)

//IResolverMock implements github.com/replicatedcom/ship/pkg/lifecycle/render/config.IResolver
type IResolverMock struct {
	t minimock.Tester

	ResolveConfigFunc       func(p context.Context) (r map[string]interface{}, r1 error)
	ResolveConfigCounter    uint64
	ResolveConfigPreCounter uint64
	ResolveConfigMock       mIResolverMockResolveConfig
}

//NewIResolverMock returns a mock for github.com/replicatedcom/ship/pkg/lifecycle/render/config.IResolver
func NewIResolverMock(t minimock.Tester) *IResolverMock {
	m := &IResolverMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.ResolveConfigMock = mIResolverMockResolveConfig{mock: m}

	return m
}

type mIResolverMockResolveConfig struct {
	mock             *IResolverMock
	mockExpectations *IResolverMockResolveConfigParams
}

//IResolverMockResolveConfigParams represents input parameters of the IResolver.ResolveConfig
type IResolverMockResolveConfigParams struct {
	p context.Context
}

//Expect sets up expected params for the IResolver.ResolveConfig
func (m *mIResolverMockResolveConfig) Expect(p context.Context) *mIResolverMockResolveConfig {
	m.mockExpectations = &IResolverMockResolveConfigParams{p}
	return m
}

//Return sets up a mock for IResolver.ResolveConfig to return Return's arguments
func (m *mIResolverMockResolveConfig) Return(r map[string]interface{}, r1 error) *IResolverMock {
	m.mock.ResolveConfigFunc = func(p context.Context) (map[string]interface{}, error) {
		return r, r1
	}
	return m.mock
}

//Set uses given function f as a mock of IResolver.ResolveConfig method
func (m *mIResolverMockResolveConfig) Set(f func(p context.Context) (r map[string]interface{}, r1 error)) *IResolverMock {
	m.mock.ResolveConfigFunc = f
	return m.mock
}

//ResolveConfig implements github.com/replicatedcom/ship/pkg/lifecycle/render/config.IResolver interface
func (m *IResolverMock) ResolveConfig(p context.Context) (r map[string]interface{}, r1 error) {
	atomic.AddUint64(&m.ResolveConfigPreCounter, 1)
	defer atomic.AddUint64(&m.ResolveConfigCounter, 1)

	if m.ResolveConfigMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.ResolveConfigMock.mockExpectations, IResolverMockResolveConfigParams{p},
			"IResolver.ResolveConfig got unexpected parameters")

		if m.ResolveConfigFunc == nil {

			m.t.Fatal("No results are set for the IResolverMock.ResolveConfig")

			return
		}
	}

	if m.ResolveConfigFunc == nil {
		m.t.Fatal("Unexpected call to IResolverMock.ResolveConfig")
		return
	}

	return m.ResolveConfigFunc(p)
}

//ResolveConfigMinimockCounter returns a count of IResolverMock.ResolveConfigFunc invocations
func (m *IResolverMock) ResolveConfigMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.ResolveConfigCounter)
}

//ResolveConfigMinimockPreCounter returns the value of IResolverMock.ResolveConfig invocations
func (m *IResolverMock) ResolveConfigMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.ResolveConfigPreCounter)
}

//ValidateCallCounters checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *IResolverMock) ValidateCallCounters() {

	if m.ResolveConfigFunc != nil && atomic.LoadUint64(&m.ResolveConfigCounter) == 0 {
		m.t.Fatal("Expected call to IResolverMock.ResolveConfig")
	}

}

//CheckMocksCalled checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *IResolverMock) CheckMocksCalled() {
	m.Finish()
}

//Finish checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish or use Finish method of minimock.Controller
func (m *IResolverMock) Finish() {
	m.MinimockFinish()
}

//MinimockFinish checks that all mocked methods of the interface have been called at least once
func (m *IResolverMock) MinimockFinish() {

	if m.ResolveConfigFunc != nil && atomic.LoadUint64(&m.ResolveConfigCounter) == 0 {
		m.t.Fatal("Expected call to IResolverMock.ResolveConfig")
	}

}

//Wait waits for all mocked methods to be called at least once
//Deprecated: please use MinimockWait or use Wait method of minimock.Controller
func (m *IResolverMock) Wait(timeout time.Duration) {
	m.MinimockWait(timeout)
}

//MinimockWait waits for all mocked methods to be called at least once
//this method is called by minimock.Controller
func (m *IResolverMock) MinimockWait(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		ok := true
		ok = ok && (m.ResolveConfigFunc == nil || atomic.LoadUint64(&m.ResolveConfigCounter) > 0)

		if ok {
			return
		}

		select {
		case <-timeoutCh:

			if m.ResolveConfigFunc != nil && atomic.LoadUint64(&m.ResolveConfigCounter) == 0 {
				m.t.Error("Expected call to IResolverMock.ResolveConfig")
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
func (m *IResolverMock) AllMocksCalled() bool {

	if m.ResolveConfigFunc != nil && atomic.LoadUint64(&m.ResolveConfigCounter) == 0 {
		return false
	}

	return true
}
