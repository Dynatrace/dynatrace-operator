// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	context "context"

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

// Reconcile provides a mock function with given fields: ctx
func (_m *Reconciler) Reconcile(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Reconciler_Reconcile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Reconcile'
type Reconciler_Reconcile_Call struct {
	*mock.Call
}

// Reconcile is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Reconciler_Expecter) Reconcile(ctx interface{}) *Reconciler_Reconcile_Call {
	return &Reconciler_Reconcile_Call{Call: _e.mock.On("Reconcile", ctx)}
}

func (_c *Reconciler_Reconcile_Call) Run(run func(ctx context.Context)) *Reconciler_Reconcile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Reconciler_Reconcile_Call) Return(_a0 error) *Reconciler_Reconcile_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Reconciler_Reconcile_Call) RunAndReturn(run func(context.Context) error) *Reconciler_Reconcile_Call {
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
