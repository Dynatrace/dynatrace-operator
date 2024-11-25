// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	context "context"

	dynatrace "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	mock "github.com/stretchr/testify/mock"

	status "github.com/Dynatrace/dynatrace-operator/pkg/api/status"
)

// StatusUpdater is an autogenerated mock type for the StatusUpdater type
type StatusUpdater struct {
	mock.Mock
}

type StatusUpdater_Expecter struct {
	mock *mock.Mock
}

func (_m *StatusUpdater) EXPECT() *StatusUpdater_Expecter {
	return &StatusUpdater_Expecter{mock: &_m.Mock}
}

// CheckForDowngrade provides a mock function with given fields: latestVersion
func (_m *StatusUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	ret := _m.Called(latestVersion)

	if len(ret) == 0 {
		panic("no return value specified for CheckForDowngrade")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(latestVersion)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(latestVersion)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(latestVersion)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StatusUpdater_CheckForDowngrade_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckForDowngrade'
type StatusUpdater_CheckForDowngrade_Call struct {
	*mock.Call
}

// CheckForDowngrade is a helper method to define mock.On call
//   - latestVersion string
func (_e *StatusUpdater_Expecter) CheckForDowngrade(latestVersion interface{}) *StatusUpdater_CheckForDowngrade_Call {
	return &StatusUpdater_CheckForDowngrade_Call{Call: _e.mock.On("CheckForDowngrade", latestVersion)}
}

func (_c *StatusUpdater_CheckForDowngrade_Call) Run(run func(latestVersion string)) *StatusUpdater_CheckForDowngrade_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *StatusUpdater_CheckForDowngrade_Call) Return(_a0 bool, _a1 error) *StatusUpdater_CheckForDowngrade_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StatusUpdater_CheckForDowngrade_Call) RunAndReturn(run func(string) (bool, error)) *StatusUpdater_CheckForDowngrade_Call {
	_c.Call.Return(run)
	return _c
}

// CustomImage provides a mock function with given fields:
func (_m *StatusUpdater) CustomImage() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CustomImage")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// StatusUpdater_CustomImage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CustomImage'
type StatusUpdater_CustomImage_Call struct {
	*mock.Call
}

// CustomImage is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) CustomImage() *StatusUpdater_CustomImage_Call {
	return &StatusUpdater_CustomImage_Call{Call: _e.mock.On("CustomImage")}
}

func (_c *StatusUpdater_CustomImage_Call) Run(run func()) *StatusUpdater_CustomImage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_CustomImage_Call) Return(_a0 string) *StatusUpdater_CustomImage_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_CustomImage_Call) RunAndReturn(run func() string) *StatusUpdater_CustomImage_Call {
	_c.Call.Return(run)
	return _c
}

// CustomVersion provides a mock function with given fields:
func (_m *StatusUpdater) CustomVersion() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CustomVersion")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// StatusUpdater_CustomVersion_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CustomVersion'
type StatusUpdater_CustomVersion_Call struct {
	*mock.Call
}

// CustomVersion is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) CustomVersion() *StatusUpdater_CustomVersion_Call {
	return &StatusUpdater_CustomVersion_Call{Call: _e.mock.On("CustomVersion")}
}

func (_c *StatusUpdater_CustomVersion_Call) Run(run func()) *StatusUpdater_CustomVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_CustomVersion_Call) Return(_a0 string) *StatusUpdater_CustomVersion_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_CustomVersion_Call) RunAndReturn(run func() string) *StatusUpdater_CustomVersion_Call {
	_c.Call.Return(run)
	return _c
}

// IsAutoUpdateEnabled provides a mock function with given fields:
func (_m *StatusUpdater) IsAutoUpdateEnabled() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsAutoUpdateEnabled")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// StatusUpdater_IsAutoUpdateEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsAutoUpdateEnabled'
type StatusUpdater_IsAutoUpdateEnabled_Call struct {
	*mock.Call
}

// IsAutoUpdateEnabled is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) IsAutoUpdateEnabled() *StatusUpdater_IsAutoUpdateEnabled_Call {
	return &StatusUpdater_IsAutoUpdateEnabled_Call{Call: _e.mock.On("IsAutoUpdateEnabled")}
}

func (_c *StatusUpdater_IsAutoUpdateEnabled_Call) Run(run func()) *StatusUpdater_IsAutoUpdateEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_IsAutoUpdateEnabled_Call) Return(_a0 bool) *StatusUpdater_IsAutoUpdateEnabled_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_IsAutoUpdateEnabled_Call) RunAndReturn(run func() bool) *StatusUpdater_IsAutoUpdateEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// IsEnabled provides a mock function with given fields:
func (_m *StatusUpdater) IsEnabled() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsEnabled")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// StatusUpdater_IsEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsEnabled'
type StatusUpdater_IsEnabled_Call struct {
	*mock.Call
}

// IsEnabled is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) IsEnabled() *StatusUpdater_IsEnabled_Call {
	return &StatusUpdater_IsEnabled_Call{Call: _e.mock.On("IsEnabled")}
}

func (_c *StatusUpdater_IsEnabled_Call) Run(run func()) *StatusUpdater_IsEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_IsEnabled_Call) Return(_a0 bool) *StatusUpdater_IsEnabled_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_IsEnabled_Call) RunAndReturn(run func() bool) *StatusUpdater_IsEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// IsPublicRegistryEnabled provides a mock function with given fields:
func (_m *StatusUpdater) IsPublicRegistryEnabled() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsPublicRegistryEnabled")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// StatusUpdater_IsPublicRegistryEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsPublicRegistryEnabled'
type StatusUpdater_IsPublicRegistryEnabled_Call struct {
	*mock.Call
}

// IsPublicRegistryEnabled is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) IsPublicRegistryEnabled() *StatusUpdater_IsPublicRegistryEnabled_Call {
	return &StatusUpdater_IsPublicRegistryEnabled_Call{Call: _e.mock.On("IsPublicRegistryEnabled")}
}

func (_c *StatusUpdater_IsPublicRegistryEnabled_Call) Run(run func()) *StatusUpdater_IsPublicRegistryEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_IsPublicRegistryEnabled_Call) Return(_a0 bool) *StatusUpdater_IsPublicRegistryEnabled_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_IsPublicRegistryEnabled_Call) RunAndReturn(run func() bool) *StatusUpdater_IsPublicRegistryEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// LatestImageInfo provides a mock function with given fields: ctx
func (_m *StatusUpdater) LatestImageInfo(ctx context.Context) (*dynatrace.LatestImageInfo, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for LatestImageInfo")
	}

	var r0 *dynatrace.LatestImageInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*dynatrace.LatestImageInfo, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *dynatrace.LatestImageInfo); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dynatrace.LatestImageInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StatusUpdater_LatestImageInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LatestImageInfo'
type StatusUpdater_LatestImageInfo_Call struct {
	*mock.Call
}

// LatestImageInfo is a helper method to define mock.On call
//   - ctx context.Context
func (_e *StatusUpdater_Expecter) LatestImageInfo(ctx interface{}) *StatusUpdater_LatestImageInfo_Call {
	return &StatusUpdater_LatestImageInfo_Call{Call: _e.mock.On("LatestImageInfo", ctx)}
}

func (_c *StatusUpdater_LatestImageInfo_Call) Run(run func(ctx context.Context)) *StatusUpdater_LatestImageInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *StatusUpdater_LatestImageInfo_Call) Return(_a0 *dynatrace.LatestImageInfo, _a1 error) *StatusUpdater_LatestImageInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StatusUpdater_LatestImageInfo_Call) RunAndReturn(run func(context.Context) (*dynatrace.LatestImageInfo, error)) *StatusUpdater_LatestImageInfo_Call {
	_c.Call.Return(run)
	return _c
}

// Name provides a mock function with given fields:
func (_m *StatusUpdater) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// StatusUpdater_Name_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Name'
type StatusUpdater_Name_Call struct {
	*mock.Call
}

// Name is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) Name() *StatusUpdater_Name_Call {
	return &StatusUpdater_Name_Call{Call: _e.mock.On("Name")}
}

func (_c *StatusUpdater_Name_Call) Run(run func()) *StatusUpdater_Name_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_Name_Call) Return(_a0 string) *StatusUpdater_Name_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_Name_Call) RunAndReturn(run func() string) *StatusUpdater_Name_Call {
	_c.Call.Return(run)
	return _c
}

// Target provides a mock function with given fields:
func (_m *StatusUpdater) Target() *status.VersionStatus {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Target")
	}

	var r0 *status.VersionStatus
	if rf, ok := ret.Get(0).(func() *status.VersionStatus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*status.VersionStatus)
		}
	}

	return r0
}

// StatusUpdater_Target_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Target'
type StatusUpdater_Target_Call struct {
	*mock.Call
}

// Target is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) Target() *StatusUpdater_Target_Call {
	return &StatusUpdater_Target_Call{Call: _e.mock.On("Target")}
}

func (_c *StatusUpdater_Target_Call) Run(run func()) *StatusUpdater_Target_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_Target_Call) Return(_a0 *status.VersionStatus) *StatusUpdater_Target_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_Target_Call) RunAndReturn(run func() *status.VersionStatus) *StatusUpdater_Target_Call {
	_c.Call.Return(run)
	return _c
}

// UseTenantRegistry provides a mock function with given fields: _a0
func (_m *StatusUpdater) UseTenantRegistry(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for UseTenantRegistry")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StatusUpdater_UseTenantRegistry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UseTenantRegistry'
type StatusUpdater_UseTenantRegistry_Call struct {
	*mock.Call
}

// UseTenantRegistry is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *StatusUpdater_Expecter) UseTenantRegistry(_a0 interface{}) *StatusUpdater_UseTenantRegistry_Call {
	return &StatusUpdater_UseTenantRegistry_Call{Call: _e.mock.On("UseTenantRegistry", _a0)}
}

func (_c *StatusUpdater_UseTenantRegistry_Call) Run(run func(_a0 context.Context)) *StatusUpdater_UseTenantRegistry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *StatusUpdater_UseTenantRegistry_Call) Return(_a0 error) *StatusUpdater_UseTenantRegistry_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_UseTenantRegistry_Call) RunAndReturn(run func(context.Context) error) *StatusUpdater_UseTenantRegistry_Call {
	_c.Call.Return(run)
	return _c
}

// ValidateStatus provides a mock function with given fields:
func (_m *StatusUpdater) ValidateStatus() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ValidateStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StatusUpdater_ValidateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateStatus'
type StatusUpdater_ValidateStatus_Call struct {
	*mock.Call
}

// ValidateStatus is a helper method to define mock.On call
func (_e *StatusUpdater_Expecter) ValidateStatus() *StatusUpdater_ValidateStatus_Call {
	return &StatusUpdater_ValidateStatus_Call{Call: _e.mock.On("ValidateStatus")}
}

func (_c *StatusUpdater_ValidateStatus_Call) Run(run func()) *StatusUpdater_ValidateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *StatusUpdater_ValidateStatus_Call) Return(_a0 error) *StatusUpdater_ValidateStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StatusUpdater_ValidateStatus_Call) RunAndReturn(run func() error) *StatusUpdater_ValidateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// NewStatusUpdater creates a new instance of StatusUpdater. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStatusUpdater(t interface {
	mock.TestingT
	Cleanup(func())
}) *StatusUpdater {
	mock := &StatusUpdater{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
