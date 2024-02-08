// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	context "context"

	dynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dynatrace "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"

	dynatraceclient "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"

	mock "github.com/stretchr/testify/mock"

	token "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
)

// Builder is an autogenerated mock type for the Builder type
type Builder struct {
	mock.Mock
}

type Builder_Expecter struct {
	mock *mock.Mock
}

func (_m *Builder) EXPECT() *Builder_Expecter {
	return &Builder_Expecter{mock: &_m.Mock}
}

// Build provides a mock function with given fields:
func (_m *Builder) Build() (dynatrace.Client, error) {
	ret := _m.Called()

	var r0 dynatrace.Client
	var r1 error
	if rf, ok := ret.Get(0).(func() (dynatrace.Client, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() dynatrace.Client); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynatrace.Client)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Builder_Build_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Build'
type Builder_Build_Call struct {
	*mock.Call
}

// Build is a helper method to define mock.On call
func (_e *Builder_Expecter) Build() *Builder_Build_Call {
	return &Builder_Build_Call{Call: _e.mock.On("Build")}
}

func (_c *Builder_Build_Call) Run(run func()) *Builder_Build_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Builder_Build_Call) Return(_a0 dynatrace.Client, _a1 error) *Builder_Build_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Builder_Build_Call) RunAndReturn(run func() (dynatrace.Client, error)) *Builder_Build_Call {
	_c.Call.Return(run)
	return _c
}

// BuildWithTokenVerification provides a mock function with given fields: dynaKubeStatus
func (_m *Builder) BuildWithTokenVerification(dynaKubeStatus *dynakube.DynaKubeStatus) (dynatrace.Client, error) {
	ret := _m.Called(dynaKubeStatus)

	var r0 dynatrace.Client
	var r1 error
	if rf, ok := ret.Get(0).(func(*dynakube.DynaKubeStatus) (dynatrace.Client, error)); ok {
		return rf(dynaKubeStatus)
	}
	if rf, ok := ret.Get(0).(func(*dynakube.DynaKubeStatus) dynatrace.Client); ok {
		r0 = rf(dynaKubeStatus)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynatrace.Client)
		}
	}

	if rf, ok := ret.Get(1).(func(*dynakube.DynaKubeStatus) error); ok {
		r1 = rf(dynaKubeStatus)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Builder_BuildWithTokenVerification_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BuildWithTokenVerification'
type Builder_BuildWithTokenVerification_Call struct {
	*mock.Call
}

// BuildWithTokenVerification is a helper method to define mock.On call
//   - dynaKubeStatus *dynakube.DynaKubeStatus
func (_e *Builder_Expecter) BuildWithTokenVerification(dynaKubeStatus interface{}) *Builder_BuildWithTokenVerification_Call {
	return &Builder_BuildWithTokenVerification_Call{Call: _e.mock.On("BuildWithTokenVerification", dynaKubeStatus)}
}

func (_c *Builder_BuildWithTokenVerification_Call) Run(run func(dynaKubeStatus *dynakube.DynaKubeStatus)) *Builder_BuildWithTokenVerification_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*dynakube.DynaKubeStatus))
	})
	return _c
}

func (_c *Builder_BuildWithTokenVerification_Call) Return(_a0 dynatrace.Client, _a1 error) *Builder_BuildWithTokenVerification_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Builder_BuildWithTokenVerification_Call) RunAndReturn(run func(*dynakube.DynaKubeStatus) (dynatrace.Client, error)) *Builder_BuildWithTokenVerification_Call {
	_c.Call.Return(run)
	return _c
}

// SetContext provides a mock function with given fields: ctx
func (_m *Builder) SetContext(ctx context.Context) dynatraceclient.Builder {
	ret := _m.Called(ctx)

	var r0 dynatraceclient.Builder
	if rf, ok := ret.Get(0).(func(context.Context) dynatraceclient.Builder); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynatraceclient.Builder)
		}
	}

	return r0
}

// Builder_SetContext_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetContext'
type Builder_SetContext_Call struct {
	*mock.Call
}

// SetContext is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Builder_Expecter) SetContext(ctx interface{}) *Builder_SetContext_Call {
	return &Builder_SetContext_Call{Call: _e.mock.On("SetContext", ctx)}
}

func (_c *Builder_SetContext_Call) Run(run func(ctx context.Context)) *Builder_SetContext_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Builder_SetContext_Call) Return(_a0 dynatraceclient.Builder) *Builder_SetContext_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Builder_SetContext_Call) RunAndReturn(run func(context.Context) dynatraceclient.Builder) *Builder_SetContext_Call {
	_c.Call.Return(run)
	return _c
}

// SetDynakube provides a mock function with given fields: _a0
func (_m *Builder) SetDynakube(_a0 dynakube.DynaKube) dynatraceclient.Builder {
	ret := _m.Called(_a0)

	var r0 dynatraceclient.Builder
	if rf, ok := ret.Get(0).(func(dynakube.DynaKube) dynatraceclient.Builder); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynatraceclient.Builder)
		}
	}

	return r0
}

// Builder_SetDynakube_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDynakube'
type Builder_SetDynakube_Call struct {
	*mock.Call
}

// SetDynakube is a helper method to define mock.On call
//   - _a0 dynakube.DynaKube
func (_e *Builder_Expecter) SetDynakube(_a0 interface{}) *Builder_SetDynakube_Call {
	return &Builder_SetDynakube_Call{Call: _e.mock.On("SetDynakube", _a0)}
}

func (_c *Builder_SetDynakube_Call) Run(run func(_a0 dynakube.DynaKube)) *Builder_SetDynakube_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(dynakube.DynaKube))
	})
	return _c
}

func (_c *Builder_SetDynakube_Call) Return(_a0 dynatraceclient.Builder) *Builder_SetDynakube_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Builder_SetDynakube_Call) RunAndReturn(run func(dynakube.DynaKube) dynatraceclient.Builder) *Builder_SetDynakube_Call {
	_c.Call.Return(run)
	return _c
}

// SetTokens provides a mock function with given fields: tokens
func (_m *Builder) SetTokens(tokens token.Tokens) dynatraceclient.Builder {
	ret := _m.Called(tokens)

	var r0 dynatraceclient.Builder
	if rf, ok := ret.Get(0).(func(token.Tokens) dynatraceclient.Builder); ok {
		r0 = rf(tokens)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynatraceclient.Builder)
		}
	}

	return r0
}

// Builder_SetTokens_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetTokens'
type Builder_SetTokens_Call struct {
	*mock.Call
}

// SetTokens is a helper method to define mock.On call
//   - tokens token.Tokens
func (_e *Builder_Expecter) SetTokens(tokens interface{}) *Builder_SetTokens_Call {
	return &Builder_SetTokens_Call{Call: _e.mock.On("SetTokens", tokens)}
}

func (_c *Builder_SetTokens_Call) Run(run func(tokens token.Tokens)) *Builder_SetTokens_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(token.Tokens))
	})
	return _c
}

func (_c *Builder_SetTokens_Call) Return(_a0 dynatraceclient.Builder) *Builder_SetTokens_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Builder_SetTokens_Call) RunAndReturn(run func(token.Tokens) dynatraceclient.Builder) *Builder_SetTokens_Call {
	_c.Call.Return(run)
	return _c
}

// NewBuilder creates a new instance of Builder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBuilder(t interface {
	mock.TestingT
	Cleanup(func())
}) *Builder {
	mock := &Builder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
