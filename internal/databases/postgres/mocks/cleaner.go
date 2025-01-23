// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/wal-g/wal-g/internal/databases/postgres (interfaces: Cleaner)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCleaner is a mock of Cleaner interface.
type MockCleaner struct {
	ctrl     *gomock.Controller
	recorder *MockCleanerMockRecorder
}

// MockCleanerMockRecorder is the mock recorder for MockCleaner.
type MockCleanerMockRecorder struct {
	mock *MockCleaner
}

// NewMockCleaner creates a new mock instance.
func NewMockCleaner(ctrl *gomock.Controller) *MockCleaner {
	mock := &MockCleaner{ctrl: ctrl}
	mock.recorder = &MockCleanerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCleaner) EXPECT() *MockCleanerMockRecorder {
	return m.recorder
}

// GetFiles mocks base method.
func (m *MockCleaner) GetFiles(arg0 string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFiles", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFiles indicates an expected call of GetFiles.
func (mr *MockCleanerMockRecorder) GetFiles(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFiles", reflect.TypeOf((*MockCleaner)(nil).GetFiles), arg0)
}

// Remove mocks base method.
func (m *MockCleaner) Remove(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Remove", arg0)
}

// Remove indicates an expected call of Remove.
func (mr *MockCleanerMockRecorder) Remove(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockCleaner)(nil).Remove), arg0)
}