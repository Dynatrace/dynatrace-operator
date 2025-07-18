// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	mock "github.com/stretchr/testify/mock"
)

// NewMutator creates a new instance of Mutator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMutator(t interface {
	mock.TestingT
	Cleanup(func())
}) *Mutator {
	mock := &Mutator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Mutator is an autogenerated mock type for the Mutator type
type Mutator struct {
	mock.Mock
}

type Mutator_Expecter struct {
	mock *mock.Mock
}

func (_m *Mutator) EXPECT() *Mutator_Expecter {
	return &Mutator_Expecter{mock: &_m.Mock}
}

// IsEnabled provides a mock function for the type Mutator
func (_mock *Mutator) IsEnabled(request *mutator.BaseRequest) bool {
	ret := _mock.Called(request)

	if len(ret) == 0 {
		panic("no return value specified for IsEnabled")
	}

	var r0 bool
	if returnFunc, ok := ret.Get(0).(func(*mutator.BaseRequest) bool); ok {
		r0 = returnFunc(request)
	} else {
		r0 = ret.Get(0).(bool)
	}
	return r0
}

// Mutator_IsEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsEnabled'
type Mutator_IsEnabled_Call struct {
	*mock.Call
}

// IsEnabled is a helper method to define mock.On call
//   - request *mutator.BaseRequest
func (_e *Mutator_Expecter) IsEnabled(request interface{}) *Mutator_IsEnabled_Call {
	return &Mutator_IsEnabled_Call{Call: _e.mock.On("IsEnabled", request)}
}

func (_c *Mutator_IsEnabled_Call) Run(run func(request *mutator.BaseRequest)) *Mutator_IsEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 *mutator.BaseRequest
		if args[0] != nil {
			arg0 = args[0].(*mutator.BaseRequest)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *Mutator_IsEnabled_Call) Return(b bool) *Mutator_IsEnabled_Call {
	_c.Call.Return(b)
	return _c
}

func (_c *Mutator_IsEnabled_Call) RunAndReturn(run func(request *mutator.BaseRequest) bool) *Mutator_IsEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// IsInjected provides a mock function for the type Mutator
func (_mock *Mutator) IsInjected(request *mutator.BaseRequest) bool {
	ret := _mock.Called(request)

	if len(ret) == 0 {
		panic("no return value specified for IsInjected")
	}

	var r0 bool
	if returnFunc, ok := ret.Get(0).(func(*mutator.BaseRequest) bool); ok {
		r0 = returnFunc(request)
	} else {
		r0 = ret.Get(0).(bool)
	}
	return r0
}

// Mutator_IsInjected_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsInjected'
type Mutator_IsInjected_Call struct {
	*mock.Call
}

// IsInjected is a helper method to define mock.On call
//   - request *mutator.BaseRequest
func (_e *Mutator_Expecter) IsInjected(request interface{}) *Mutator_IsInjected_Call {
	return &Mutator_IsInjected_Call{Call: _e.mock.On("IsInjected", request)}
}

func (_c *Mutator_IsInjected_Call) Run(run func(request *mutator.BaseRequest)) *Mutator_IsInjected_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 *mutator.BaseRequest
		if args[0] != nil {
			arg0 = args[0].(*mutator.BaseRequest)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *Mutator_IsInjected_Call) Return(b bool) *Mutator_IsInjected_Call {
	_c.Call.Return(b)
	return _c
}

func (_c *Mutator_IsInjected_Call) RunAndReturn(run func(request *mutator.BaseRequest) bool) *Mutator_IsInjected_Call {
	_c.Call.Return(run)
	return _c
}

// Mutate provides a mock function for the type Mutator
func (_mock *Mutator) Mutate(request *mutator.MutationRequest) error {
	ret := _mock.Called(request)

	if len(ret) == 0 {
		panic("no return value specified for Mutate")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(*mutator.MutationRequest) error); ok {
		r0 = returnFunc(request)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// Mutator_Mutate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Mutate'
type Mutator_Mutate_Call struct {
	*mock.Call
}

// Mutate is a helper method to define mock.On call
//   - request *mutator.MutationRequest
func (_e *Mutator_Expecter) Mutate(request interface{}) *Mutator_Mutate_Call {
	return &Mutator_Mutate_Call{Call: _e.mock.On("Mutate", request)}
}

func (_c *Mutator_Mutate_Call) Run(run func(request *mutator.MutationRequest)) *Mutator_Mutate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 *mutator.MutationRequest
		if args[0] != nil {
			arg0 = args[0].(*mutator.MutationRequest)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *Mutator_Mutate_Call) Return(err error) *Mutator_Mutate_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *Mutator_Mutate_Call) RunAndReturn(run func(request *mutator.MutationRequest) error) *Mutator_Mutate_Call {
	_c.Call.Return(run)
	return _c
}

// Reinvoke provides a mock function for the type Mutator
func (_mock *Mutator) Reinvoke(request *mutator.ReinvocationRequest) bool {
	ret := _mock.Called(request)

	if len(ret) == 0 {
		panic("no return value specified for Reinvoke")
	}

	var r0 bool
	if returnFunc, ok := ret.Get(0).(func(*mutator.ReinvocationRequest) bool); ok {
		r0 = returnFunc(request)
	} else {
		r0 = ret.Get(0).(bool)
	}
	return r0
}

// Mutator_Reinvoke_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Reinvoke'
type Mutator_Reinvoke_Call struct {
	*mock.Call
}

// Reinvoke is a helper method to define mock.On call
//   - request *mutator.ReinvocationRequest
func (_e *Mutator_Expecter) Reinvoke(request interface{}) *Mutator_Reinvoke_Call {
	return &Mutator_Reinvoke_Call{Call: _e.mock.On("Reinvoke", request)}
}

func (_c *Mutator_Reinvoke_Call) Run(run func(request *mutator.ReinvocationRequest)) *Mutator_Reinvoke_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 *mutator.ReinvocationRequest
		if args[0] != nil {
			arg0 = args[0].(*mutator.ReinvocationRequest)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *Mutator_Reinvoke_Call) Return(b bool) *Mutator_Reinvoke_Call {
	_c.Call.Return(b)
	return _c
}

func (_c *Mutator_Reinvoke_Call) RunAndReturn(run func(request *mutator.ReinvocationRequest) bool) *Mutator_Reinvoke_Call {
	_c.Call.Return(run)
	return _c
}
