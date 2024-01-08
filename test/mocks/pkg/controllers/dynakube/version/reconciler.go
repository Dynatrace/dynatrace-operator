// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	context "context"

	dynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	mock "github.com/stretchr/testify/mock"
)

// Reconciler is an autogenerated mock type for the Reconciler type
type Reconciler struct {
	mock.Mock
}

type Reconciler_Expecter struct {
	mock *mock.Mock
}

func (_m *Reconciler) EXPECT() *Reconciler_Expecter {
	return &Reconciler_Expecter{mock: &_m.Mock}
}

// ReconcileActiveGate provides a mock function with given fields: ctx, _a1
func (_m *Reconciler) ReconcileActiveGate(ctx context.Context, _a1 *dynakube.DynaKube) error {
	ret := _m.Called(ctx, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *dynakube.DynaKube) error); ok {
		r0 = rf(ctx, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Reconciler_ReconcileActiveGate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReconcileActiveGate'
type Reconciler_ReconcileActiveGate_Call struct {
	*mock.Call
}

// ReconcileActiveGate is a helper method to define mock.On call
//   - ctx context.Context
//   - _a1 *dynakube.DynaKube
func (_e *Reconciler_Expecter) ReconcileActiveGate(ctx interface{}, _a1 interface{}) *Reconciler_ReconcileActiveGate_Call {
	return &Reconciler_ReconcileActiveGate_Call{Call: _e.mock.On("ReconcileActiveGate", ctx, _a1)}
}

func (_c *Reconciler_ReconcileActiveGate_Call) Run(run func(ctx context.Context, _a1 *dynakube.DynaKube)) *Reconciler_ReconcileActiveGate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*dynakube.DynaKube))
	})
	return _c
}

func (_c *Reconciler_ReconcileActiveGate_Call) Return(_a0 error) *Reconciler_ReconcileActiveGate_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Reconciler_ReconcileActiveGate_Call) RunAndReturn(run func(context.Context, *dynakube.DynaKube) error) *Reconciler_ReconcileActiveGate_Call {
	_c.Call.Return(run)
	return _c
}

// ReconcileCodeModules provides a mock function with given fields: ctx, _a1
func (_m *Reconciler) ReconcileCodeModules(ctx context.Context, _a1 *dynakube.DynaKube) error {
	ret := _m.Called(ctx, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *dynakube.DynaKube) error); ok {
		r0 = rf(ctx, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Reconciler_ReconcileCodeModules_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReconcileCodeModules'
type Reconciler_ReconcileCodeModules_Call struct {
	*mock.Call
}

// ReconcileCodeModules is a helper method to define mock.On call
//   - ctx context.Context
//   - _a1 *dynakube.DynaKube
func (_e *Reconciler_Expecter) ReconcileCodeModules(ctx interface{}, _a1 interface{}) *Reconciler_ReconcileCodeModules_Call {
	return &Reconciler_ReconcileCodeModules_Call{Call: _e.mock.On("ReconcileCodeModules", ctx, _a1)}
}

func (_c *Reconciler_ReconcileCodeModules_Call) Run(run func(ctx context.Context, _a1 *dynakube.DynaKube)) *Reconciler_ReconcileCodeModules_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*dynakube.DynaKube))
	})
	return _c
}

func (_c *Reconciler_ReconcileCodeModules_Call) Return(_a0 error) *Reconciler_ReconcileCodeModules_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Reconciler_ReconcileCodeModules_Call) RunAndReturn(run func(context.Context, *dynakube.DynaKube) error) *Reconciler_ReconcileCodeModules_Call {
	_c.Call.Return(run)
	return _c
}

// ReconcileOneAgent provides a mock function with given fields: ctx, _a1
func (_m *Reconciler) ReconcileOneAgent(ctx context.Context, _a1 *dynakube.DynaKube) error {
	ret := _m.Called(ctx, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *dynakube.DynaKube) error); ok {
		r0 = rf(ctx, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Reconciler_ReconcileOneAgent_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReconcileOneAgent'
type Reconciler_ReconcileOneAgent_Call struct {
	*mock.Call
}

// ReconcileOneAgent is a helper method to define mock.On call
//   - ctx context.Context
//   - _a1 *dynakube.DynaKube
func (_e *Reconciler_Expecter) ReconcileOneAgent(ctx interface{}, _a1 interface{}) *Reconciler_ReconcileOneAgent_Call {
	return &Reconciler_ReconcileOneAgent_Call{Call: _e.mock.On("ReconcileOneAgent", ctx, _a1)}
}

func (_c *Reconciler_ReconcileOneAgent_Call) Run(run func(ctx context.Context, _a1 *dynakube.DynaKube)) *Reconciler_ReconcileOneAgent_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*dynakube.DynaKube))
	})
	return _c
}

func (_c *Reconciler_ReconcileOneAgent_Call) Return(_a0 error) *Reconciler_ReconcileOneAgent_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Reconciler_ReconcileOneAgent_Call) RunAndReturn(run func(context.Context, *dynakube.DynaKube) error) *Reconciler_ReconcileOneAgent_Call {
	_c.Call.Return(run)
	return _c
}

// NewReconciler creates a new instance of Reconciler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReconciler(t interface {
	mock.TestingT
	Cleanup(func())
}) *Reconciler {
	mock := &Reconciler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
