// Code generated by mockery v2.32.4. DO NOT EDIT.

package mocks

import (
	context "context"
	http "net/http"

	registry "github.com/Dynatrace/dynatrace-operator/src/registry"
	authn "github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	mock "github.com/stretchr/testify/mock"
)

// MockImageGetter is an autogenerated mock type for the ImageGetter type
type MockImageGetter struct {
	mock.Mock
}

// GetImageVersion provides a mock function with given fields: ctx, keychain, transport, imageName
func (_m *MockImageGetter) GetImageVersion(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (registry.ImageVersion, error) {
	ret := _m.Called(ctx, keychain, transport, imageName)

	var r0 registry.ImageVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, authn.Keychain, *http.Transport, string) (registry.ImageVersion, error)); ok {
		return rf(ctx, keychain, transport, imageName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, authn.Keychain, *http.Transport, string) registry.ImageVersion); ok {
		r0 = rf(ctx, keychain, transport, imageName)
	} else {
		r0 = ret.Get(0).(registry.ImageVersion)
	}

	if rf, ok := ret.Get(1).(func(context.Context, authn.Keychain, *http.Transport, string) error); ok {
		r1 = rf(ctx, keychain, transport, imageName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PullImageInfo provides a mock function with given fields: ctx, keychain, transport, imageName
func (_m *MockImageGetter) PullImageInfo(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (*v1.Image, error) {
	ret := _m.Called(ctx, keychain, transport, imageName)

	var r0 *v1.Image
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, authn.Keychain, *http.Transport, string) (*v1.Image, error)); ok {
		return rf(ctx, keychain, transport, imageName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, authn.Keychain, *http.Transport, string) *v1.Image); ok {
		r0 = rf(ctx, keychain, transport, imageName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Image)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, authn.Keychain, *http.Transport, string) error); ok {
		r1 = rf(ctx, keychain, transport, imageName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockImageGetter creates a new instance of MockImageGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockImageGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockImageGetter {
	mock := &MockImageGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
