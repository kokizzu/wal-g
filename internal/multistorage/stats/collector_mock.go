// Code generated by MockGen. DO NOT EDIT.
// Source: collector.go

// Package stats is a generated GoMock package.
package stats

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCollector is a mock of Collector interface.
type MockCollector struct {
	ctrl     *gomock.Controller
	recorder *MockCollectorMockRecorder
}

// MockCollectorMockRecorder is the mock recorder for MockCollector.
type MockCollectorMockRecorder struct {
	mock *MockCollector
}

// NewMockCollector creates a new mock instance.
func NewMockCollector(ctrl *gomock.Controller) *MockCollector {
	mock := &MockCollector{ctrl: ctrl}
	mock.recorder = &MockCollectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCollector) EXPECT() *MockCollectorMockRecorder {
	return m.recorder
}

// AllAliveStorages mocks base method.
func (m *MockCollector) AllAliveStorages() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllAliveStorages")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AllAliveStorages indicates an expected call of AllAliveStorages.
func (mr *MockCollectorMockRecorder) AllAliveStorages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllAliveStorages", reflect.TypeOf((*MockCollector)(nil).AllAliveStorages))
}

// Close mocks base method.
func (m *MockCollector) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockCollectorMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockCollector)(nil).Close))
}

// FirstAliveStorage mocks base method.
func (m *MockCollector) FirstAliveStorage() (*string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FirstAliveStorage")
	ret0, _ := ret[0].(*string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FirstAliveStorage indicates an expected call of FirstAliveStorage.
func (mr *MockCollectorMockRecorder) FirstAliveStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FirstAliveStorage", reflect.TypeOf((*MockCollector)(nil).FirstAliveStorage))
}

// ReportOperationResult mocks base method.
func (m *MockCollector) ReportOperationResult(storage string, op OperationWeight, success bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReportOperationResult", storage, op, success)
}

// ReportOperationResult indicates an expected call of ReportOperationResult.
func (mr *MockCollectorMockRecorder) ReportOperationResult(storage, op, success interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportOperationResult", reflect.TypeOf((*MockCollector)(nil).ReportOperationResult), storage, op, success)
}

// SpecificStorage mocks base method.
func (m *MockCollector) SpecificStorage(name string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpecificStorage", name)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SpecificStorage indicates an expected call of SpecificStorage.
func (mr *MockCollectorMockRecorder) SpecificStorage(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpecificStorage", reflect.TypeOf((*MockCollector)(nil).SpecificStorage), name)
}