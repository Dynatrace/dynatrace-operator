// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	edgeconnect "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	logd "github.com/Dynatrace/dynatrace-operator/pkg/logd"

	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

type Client_Expecter struct {
	mock *mock.Mock
}

func (_m *Client) EXPECT() *Client_Expecter {
	return &Client_Expecter{mock: &_m.Mock}
}

// CreateConnectionSetting provides a mock function with given fields: es
func (_m *Client) CreateConnectionSetting(es edgeconnect.EnvironmentSetting) error {
	ret := _m.Called(es)

	if len(ret) == 0 {
		panic("no return value specified for CreateConnectionSetting")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(edgeconnect.EnvironmentSetting) error); ok {
		r0 = rf(es)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_CreateConnectionSetting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateConnectionSetting'
type Client_CreateConnectionSetting_Call struct {
	*mock.Call
}

// CreateConnectionSetting is a helper method to define mock.On call
//   - es edgeconnect.EnvironmentSetting
func (_e *Client_Expecter) CreateConnectionSetting(es interface{}) *Client_CreateConnectionSetting_Call {
	return &Client_CreateConnectionSetting_Call{Call: _e.mock.On("CreateConnectionSetting", es)}
}

func (_c *Client_CreateConnectionSetting_Call) Run(run func(es edgeconnect.EnvironmentSetting)) *Client_CreateConnectionSetting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(edgeconnect.EnvironmentSetting))
	})
	return _c
}

func (_c *Client_CreateConnectionSetting_Call) Return(_a0 error) *Client_CreateConnectionSetting_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_CreateConnectionSetting_Call) RunAndReturn(run func(edgeconnect.EnvironmentSetting) error) *Client_CreateConnectionSetting_Call {
	_c.Call.Return(run)
	return _c
}

// CreateEdgeConnect provides a mock function with given fields: name, hostPatterns, oauthClientId
func (_m *Client) CreateEdgeConnect(name string, hostPatterns []string, oauthClientId string) (edgeconnect.CreateResponse, error) {
	ret := _m.Called(name, hostPatterns, oauthClientId)

	if len(ret) == 0 {
		panic("no return value specified for CreateEdgeConnect")
	}

	var r0 edgeconnect.CreateResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string, []string, string) (edgeconnect.CreateResponse, error)); ok {
		return rf(name, hostPatterns, oauthClientId)
	}
	if rf, ok := ret.Get(0).(func(string, []string, string) edgeconnect.CreateResponse); ok {
		r0 = rf(name, hostPatterns, oauthClientId)
	} else {
		r0 = ret.Get(0).(edgeconnect.CreateResponse)
	}

	if rf, ok := ret.Get(1).(func(string, []string, string) error); ok {
		r1 = rf(name, hostPatterns, oauthClientId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_CreateEdgeConnect_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateEdgeConnect'
type Client_CreateEdgeConnect_Call struct {
	*mock.Call
}

// CreateEdgeConnect is a helper method to define mock.On call
//   - name string
//   - hostPatterns []string
//   - oauthClientId string
func (_e *Client_Expecter) CreateEdgeConnect(name interface{}, hostPatterns interface{}, oauthClientId interface{}) *Client_CreateEdgeConnect_Call {
	return &Client_CreateEdgeConnect_Call{Call: _e.mock.On("CreateEdgeConnect", name, hostPatterns, oauthClientId)}
}

func (_c *Client_CreateEdgeConnect_Call) Run(run func(name string, hostPatterns []string, oauthClientId string)) *Client_CreateEdgeConnect_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].([]string), args[2].(string))
	})
	return _c
}

func (_c *Client_CreateEdgeConnect_Call) Return(_a0 edgeconnect.CreateResponse, _a1 error) *Client_CreateEdgeConnect_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_CreateEdgeConnect_Call) RunAndReturn(run func(string, []string, string) (edgeconnect.CreateResponse, error)) *Client_CreateEdgeConnect_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteConnectionSetting provides a mock function with given fields: objectId
func (_m *Client) DeleteConnectionSetting(objectId string) error {
	ret := _m.Called(objectId)

	if len(ret) == 0 {
		panic("no return value specified for DeleteConnectionSetting")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(objectId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_DeleteConnectionSetting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteConnectionSetting'
type Client_DeleteConnectionSetting_Call struct {
	*mock.Call
}

// DeleteConnectionSetting is a helper method to define mock.On call
//   - objectId string
func (_e *Client_Expecter) DeleteConnectionSetting(objectId interface{}) *Client_DeleteConnectionSetting_Call {
	return &Client_DeleteConnectionSetting_Call{Call: _e.mock.On("DeleteConnectionSetting", objectId)}
}

func (_c *Client_DeleteConnectionSetting_Call) Run(run func(objectId string)) *Client_DeleteConnectionSetting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Client_DeleteConnectionSetting_Call) Return(_a0 error) *Client_DeleteConnectionSetting_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_DeleteConnectionSetting_Call) RunAndReturn(run func(string) error) *Client_DeleteConnectionSetting_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteEdgeConnect provides a mock function with given fields: edgeConnectId
func (_m *Client) DeleteEdgeConnect(edgeConnectId string) error {
	ret := _m.Called(edgeConnectId)

	if len(ret) == 0 {
		panic("no return value specified for DeleteEdgeConnect")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(edgeConnectId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_DeleteEdgeConnect_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteEdgeConnect'
type Client_DeleteEdgeConnect_Call struct {
	*mock.Call
}

// DeleteEdgeConnect is a helper method to define mock.On call
//   - edgeConnectId string
func (_e *Client_Expecter) DeleteEdgeConnect(edgeConnectId interface{}) *Client_DeleteEdgeConnect_Call {
	return &Client_DeleteEdgeConnect_Call{Call: _e.mock.On("DeleteEdgeConnect", edgeConnectId)}
}

func (_c *Client_DeleteEdgeConnect_Call) Run(run func(edgeConnectId string)) *Client_DeleteEdgeConnect_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Client_DeleteEdgeConnect_Call) Return(_a0 error) *Client_DeleteEdgeConnect_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_DeleteEdgeConnect_Call) RunAndReturn(run func(string) error) *Client_DeleteEdgeConnect_Call {
	_c.Call.Return(run)
	return _c
}

// GetConnectionSetting provides a mock function with given fields: log
func (_m *Client) GetConnectionSetting(log *logd.Logger) (edgeconnect.EnvironmentSetting, error) {
	ret := _m.Called(log)

	if len(ret) == 0 {
		panic("no return value specified for GetConnectionSetting")
	}

	var r0 edgeconnect.EnvironmentSetting
	var r1 error
	if rf, ok := ret.Get(0).(func(*logd.Logger) (edgeconnect.EnvironmentSetting, error)); ok {
		return rf(log)
	}
	if rf, ok := ret.Get(0).(func(*logd.Logger) edgeconnect.EnvironmentSetting); ok {
		r0 = rf(log)
	} else {
		r0 = ret.Get(0).(edgeconnect.EnvironmentSetting)
	}

	if rf, ok := ret.Get(1).(func(*logd.Logger) error); ok {
		r1 = rf(log)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetConnectionSetting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConnectionSetting'
type Client_GetConnectionSetting_Call struct {
	*mock.Call
}

// GetConnectionSetting is a helper method to define mock.On call
//   - log *logd.Logger
func (_e *Client_Expecter) GetConnectionSetting(log interface{}) *Client_GetConnectionSetting_Call {
	return &Client_GetConnectionSetting_Call{Call: _e.mock.On("GetConnectionSetting", log)}
}

func (_c *Client_GetConnectionSetting_Call) Run(run func(log *logd.Logger)) *Client_GetConnectionSetting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*logd.Logger))
	})
	return _c
}

func (_c *Client_GetConnectionSetting_Call) Return(_a0 edgeconnect.EnvironmentSetting, _a1 error) *Client_GetConnectionSetting_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetConnectionSetting_Call) RunAndReturn(run func(*logd.Logger) (edgeconnect.EnvironmentSetting, error)) *Client_GetConnectionSetting_Call {
	_c.Call.Return(run)
	return _c
}

// GetEdgeConnect provides a mock function with given fields: edgeConnectId
func (_m *Client) GetEdgeConnect(edgeConnectId string) (edgeconnect.GetResponse, error) {
	ret := _m.Called(edgeConnectId)

	if len(ret) == 0 {
		panic("no return value specified for GetEdgeConnect")
	}

	var r0 edgeconnect.GetResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (edgeconnect.GetResponse, error)); ok {
		return rf(edgeConnectId)
	}
	if rf, ok := ret.Get(0).(func(string) edgeconnect.GetResponse); ok {
		r0 = rf(edgeConnectId)
	} else {
		r0 = ret.Get(0).(edgeconnect.GetResponse)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(edgeConnectId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetEdgeConnect_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetEdgeConnect'
type Client_GetEdgeConnect_Call struct {
	*mock.Call
}

// GetEdgeConnect is a helper method to define mock.On call
//   - edgeConnectId string
func (_e *Client_Expecter) GetEdgeConnect(edgeConnectId interface{}) *Client_GetEdgeConnect_Call {
	return &Client_GetEdgeConnect_Call{Call: _e.mock.On("GetEdgeConnect", edgeConnectId)}
}

func (_c *Client_GetEdgeConnect_Call) Run(run func(edgeConnectId string)) *Client_GetEdgeConnect_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Client_GetEdgeConnect_Call) Return(_a0 edgeconnect.GetResponse, _a1 error) *Client_GetEdgeConnect_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetEdgeConnect_Call) RunAndReturn(run func(string) (edgeconnect.GetResponse, error)) *Client_GetEdgeConnect_Call {
	_c.Call.Return(run)
	return _c
}

// GetEdgeConnects provides a mock function with given fields: name
func (_m *Client) GetEdgeConnects(name string) (edgeconnect.ListResponse, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for GetEdgeConnects")
	}

	var r0 edgeconnect.ListResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (edgeconnect.ListResponse, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) edgeconnect.ListResponse); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Get(0).(edgeconnect.ListResponse)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetEdgeConnects_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetEdgeConnects'
type Client_GetEdgeConnects_Call struct {
	*mock.Call
}

// GetEdgeConnects is a helper method to define mock.On call
//   - name string
func (_e *Client_Expecter) GetEdgeConnects(name interface{}) *Client_GetEdgeConnects_Call {
	return &Client_GetEdgeConnects_Call{Call: _e.mock.On("GetEdgeConnects", name)}
}

func (_c *Client_GetEdgeConnects_Call) Run(run func(name string)) *Client_GetEdgeConnects_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Client_GetEdgeConnects_Call) Return(_a0 edgeconnect.ListResponse, _a1 error) *Client_GetEdgeConnects_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetEdgeConnects_Call) RunAndReturn(run func(string) (edgeconnect.ListResponse, error)) *Client_GetEdgeConnects_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateConnectionSetting provides a mock function with given fields: es
func (_m *Client) UpdateConnectionSetting(es edgeconnect.EnvironmentSetting) error {
	ret := _m.Called(es)

	if len(ret) == 0 {
		panic("no return value specified for UpdateConnectionSetting")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(edgeconnect.EnvironmentSetting) error); ok {
		r0 = rf(es)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_UpdateConnectionSetting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateConnectionSetting'
type Client_UpdateConnectionSetting_Call struct {
	*mock.Call
}

// UpdateConnectionSetting is a helper method to define mock.On call
//   - es edgeconnect.EnvironmentSetting
func (_e *Client_Expecter) UpdateConnectionSetting(es interface{}) *Client_UpdateConnectionSetting_Call {
	return &Client_UpdateConnectionSetting_Call{Call: _e.mock.On("UpdateConnectionSetting", es)}
}

func (_c *Client_UpdateConnectionSetting_Call) Run(run func(es edgeconnect.EnvironmentSetting)) *Client_UpdateConnectionSetting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(edgeconnect.EnvironmentSetting))
	})
	return _c
}

func (_c *Client_UpdateConnectionSetting_Call) Return(_a0 error) *Client_UpdateConnectionSetting_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_UpdateConnectionSetting_Call) RunAndReturn(run func(edgeconnect.EnvironmentSetting) error) *Client_UpdateConnectionSetting_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateEdgeConnect provides a mock function with given fields: edgeConnectId, name, hostPatterns, oauthClientId
func (_m *Client) UpdateEdgeConnect(edgeConnectId string, name string, hostPatterns []string, oauthClientId string) error {
	ret := _m.Called(edgeConnectId, name, hostPatterns, oauthClientId)

	if len(ret) == 0 {
		panic("no return value specified for UpdateEdgeConnect")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, []string, string) error); ok {
		r0 = rf(edgeConnectId, name, hostPatterns, oauthClientId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_UpdateEdgeConnect_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateEdgeConnect'
type Client_UpdateEdgeConnect_Call struct {
	*mock.Call
}

// UpdateEdgeConnect is a helper method to define mock.On call
//   - edgeConnectId string
//   - name string
//   - hostPatterns []string
//   - oauthClientId string
func (_e *Client_Expecter) UpdateEdgeConnect(edgeConnectId interface{}, name interface{}, hostPatterns interface{}, oauthClientId interface{}) *Client_UpdateEdgeConnect_Call {
	return &Client_UpdateEdgeConnect_Call{Call: _e.mock.On("UpdateEdgeConnect", edgeConnectId, name, hostPatterns, oauthClientId)}
}

func (_c *Client_UpdateEdgeConnect_Call) Run(run func(edgeConnectId string, name string, hostPatterns []string, oauthClientId string)) *Client_UpdateEdgeConnect_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].([]string), args[3].(string))
	})
	return _c
}

func (_c *Client_UpdateEdgeConnect_Call) Return(_a0 error) *Client_UpdateEdgeConnect_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_UpdateEdgeConnect_Call) RunAndReturn(run func(string, string, []string, string) error) *Client_UpdateEdgeConnect_Call {
	_c.Call.Return(run)
	return _c
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
