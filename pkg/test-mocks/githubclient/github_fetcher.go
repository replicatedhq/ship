// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/replicatedhq/ship/pkg/specs/githubclient (interfaces: GitHubFetcher)

// Package githubclient is a generated GoMock package.
package githubclient

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockGitHubFetcher is a mock of GitHubFetcher interface
type MockGitHubFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockGitHubFetcherMockRecorder
}

// MockGitHubFetcherMockRecorder is the mock recorder for MockGitHubFetcher
type MockGitHubFetcherMockRecorder struct {
	mock *MockGitHubFetcher
}

// NewMockGitHubFetcher creates a new mock instance
func NewMockGitHubFetcher(ctrl *gomock.Controller) *MockGitHubFetcher {
	mock := &MockGitHubFetcher{ctrl: ctrl}
	mock.recorder = &MockGitHubFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGitHubFetcher) EXPECT() *MockGitHubFetcherMockRecorder {
	return m.recorder
}

// ResolveLatestRelease mocks base method
func (m *MockGitHubFetcher) ResolveLatestRelease(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveLatestRelease", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveLatestRelease indicates an expected call of ResolveLatestRelease
func (mr *MockGitHubFetcherMockRecorder) ResolveLatestRelease(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveLatestRelease", reflect.TypeOf((*MockGitHubFetcher)(nil).ResolveLatestRelease), arg0, arg1)
}

// ResolveReleaseNotes mocks base method
func (m *MockGitHubFetcher) ResolveReleaseNotes(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveReleaseNotes", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveReleaseNotes indicates an expected call of ResolveReleaseNotes
func (mr *MockGitHubFetcherMockRecorder) ResolveReleaseNotes(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveReleaseNotes", reflect.TypeOf((*MockGitHubFetcher)(nil).ResolveReleaseNotes), arg0, arg1)
}
