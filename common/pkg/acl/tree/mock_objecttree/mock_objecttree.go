// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/tree (interfaces: ObjectTree)

// Package mock_tree is a generated GoMock package.
package mock_tree

import (
	context "context"
	reflect "reflect"

	storage "github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/storage"
	tree "github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/tree"
	treechangeproto "github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/treechangeproto"
	gomock "github.com/golang/mock/gomock"
)

// MockObjectTree is a mock of ObjectTree interface.
type MockObjectTree struct {
	ctrl     *gomock.Controller
	recorder *MockObjectTreeMockRecorder
}

// MockObjectTreeMockRecorder is the mock recorder for MockObjectTree.
type MockObjectTreeMockRecorder struct {
	mock *MockObjectTree
}

// NewMockObjectTree creates a new mock instance.
func NewMockObjectTree(ctrl *gomock.Controller) *MockObjectTree {
	mock := &MockObjectTree{ctrl: ctrl}
	mock.recorder = &MockObjectTreeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockObjectTree) EXPECT() *MockObjectTreeMockRecorder {
	return m.recorder
}

// AddContent mocks base method.
func (m *MockObjectTree) AddContent(arg0 context.Context, arg1 tree.SignableChangeContent) (tree.AddResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddContent", arg0, arg1)
	ret0, _ := ret[0].(tree.AddResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddContent indicates an expected call of AddContent.
func (mr *MockObjectTreeMockRecorder) AddContent(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddContent", reflect.TypeOf((*MockObjectTree)(nil).AddContent), arg0, arg1)
}

// AddRawChanges mocks base method.
func (m *MockObjectTree) AddRawChanges(arg0 context.Context, arg1 ...*treechangeproto.RawTreeChangeWithId) (tree.AddResult, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "AddRawChanges", varargs...)
	ret0, _ := ret[0].(tree.AddResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddRawChanges indicates an expected call of AddRawChanges.
func (mr *MockObjectTreeMockRecorder) AddRawChanges(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddRawChanges", reflect.TypeOf((*MockObjectTree)(nil).AddRawChanges), varargs...)
}

// ChangesAfterCommonSnapshot mocks base method.
func (m *MockObjectTree) ChangesAfterCommonSnapshot(arg0, arg1 []string) ([]*treechangeproto.RawTreeChangeWithId, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangesAfterCommonSnapshot", arg0, arg1)
	ret0, _ := ret[0].([]*treechangeproto.RawTreeChangeWithId)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChangesAfterCommonSnapshot indicates an expected call of ChangesAfterCommonSnapshot.
func (mr *MockObjectTreeMockRecorder) ChangesAfterCommonSnapshot(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangesAfterCommonSnapshot", reflect.TypeOf((*MockObjectTree)(nil).ChangesAfterCommonSnapshot), arg0, arg1)
}

// Close mocks base method.
func (m *MockObjectTree) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockObjectTreeMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockObjectTree)(nil).Close))
}

// DebugDump mocks base method.
func (m *MockObjectTree) DebugDump() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DebugDump")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DebugDump indicates an expected call of DebugDump.
func (mr *MockObjectTreeMockRecorder) DebugDump() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DebugDump", reflect.TypeOf((*MockObjectTree)(nil).DebugDump))
}

// HasChanges mocks base method.
func (m *MockObjectTree) HasChanges(arg0 ...string) bool {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "HasChanges", varargs...)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasChanges indicates an expected call of HasChanges.
func (mr *MockObjectTreeMockRecorder) HasChanges(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasChanges", reflect.TypeOf((*MockObjectTree)(nil).HasChanges), arg0...)
}

// Header mocks base method.
func (m *MockObjectTree) Header() *treechangeproto.RawTreeChangeWithId {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(*treechangeproto.RawTreeChangeWithId)
	return ret0
}

// Header indicates an expected call of Header.
func (mr *MockObjectTreeMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockObjectTree)(nil).Header))
}

// Heads mocks base method.
func (m *MockObjectTree) Heads() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Heads")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Heads indicates an expected call of Heads.
func (mr *MockObjectTreeMockRecorder) Heads() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Heads", reflect.TypeOf((*MockObjectTree)(nil).Heads))
}

// ID mocks base method.
func (m *MockObjectTree) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockObjectTreeMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockObjectTree)(nil).ID))
}

// Iterate mocks base method.
func (m *MockObjectTree) Iterate(arg0 func([]byte) (interface{}, error), arg1 func(*tree.Change) bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Iterate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Iterate indicates an expected call of Iterate.
func (mr *MockObjectTreeMockRecorder) Iterate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Iterate", reflect.TypeOf((*MockObjectTree)(nil).Iterate), arg0, arg1)
}

// IterateFrom mocks base method.
func (m *MockObjectTree) IterateFrom(arg0 string, arg1 func([]byte) (interface{}, error), arg2 func(*tree.Change) bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IterateFrom", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// IterateFrom indicates an expected call of IterateFrom.
func (mr *MockObjectTreeMockRecorder) IterateFrom(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateFrom", reflect.TypeOf((*MockObjectTree)(nil).IterateFrom), arg0, arg1, arg2)
}

// Lock mocks base method.
func (m *MockObjectTree) Lock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Lock")
}

// Lock indicates an expected call of Lock.
func (mr *MockObjectTreeMockRecorder) Lock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lock", reflect.TypeOf((*MockObjectTree)(nil).Lock))
}

// RLock mocks base method.
func (m *MockObjectTree) RLock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RLock")
}

// RLock indicates an expected call of RLock.
func (mr *MockObjectTreeMockRecorder) RLock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RLock", reflect.TypeOf((*MockObjectTree)(nil).RLock))
}

// RUnlock mocks base method.
func (m *MockObjectTree) RUnlock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RUnlock")
}

// RUnlock indicates an expected call of RUnlock.
func (mr *MockObjectTreeMockRecorder) RUnlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RUnlock", reflect.TypeOf((*MockObjectTree)(nil).RUnlock))
}

// Root mocks base method.
func (m *MockObjectTree) Root() *tree.Change {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Root")
	ret0, _ := ret[0].(*tree.Change)
	return ret0
}

// Root indicates an expected call of Root.
func (mr *MockObjectTreeMockRecorder) Root() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Root", reflect.TypeOf((*MockObjectTree)(nil).Root))
}

// SnapshotPath mocks base method.
func (m *MockObjectTree) SnapshotPath() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SnapshotPath")
	ret0, _ := ret[0].([]string)
	return ret0
}

// SnapshotPath indicates an expected call of SnapshotPath.
func (mr *MockObjectTreeMockRecorder) SnapshotPath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SnapshotPath", reflect.TypeOf((*MockObjectTree)(nil).SnapshotPath))
}

// Storage mocks base method.
func (m *MockObjectTree) Storage() storage.TreeStorage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Storage")
	ret0, _ := ret[0].(storage.TreeStorage)
	return ret0
}

// Storage indicates an expected call of Storage.
func (mr *MockObjectTreeMockRecorder) Storage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Storage", reflect.TypeOf((*MockObjectTree)(nil).Storage))
}

// Unlock mocks base method.
func (m *MockObjectTree) Unlock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unlock")
}

// Unlock indicates an expected call of Unlock.
func (mr *MockObjectTreeMockRecorder) Unlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unlock", reflect.TypeOf((*MockObjectTree)(nil).Unlock))
}
