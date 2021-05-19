package dtclient

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockDynatraceClient implements a Dynatrace REST API Client mock
type MockDynatraceClient struct {
	mock.Mock
}

func (o *MockDynatraceClient) GetAGTenantInfo() (*TenantInfo, error) {
	args := o.Called()
	return args.Get(0).(*TenantInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetAgentTenantInfo() (*TenantInfo, error) {
	args := o.Called()
	return args.Get(0).(*TenantInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgent(os, installerType, flavor, arch string) (io.ReadCloser, error) {
	args := o.Called(os, installerType, flavor, arch)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (o *MockDynatraceClient) GetConnectionInfo() (ConnectionInfo, error) {
	args := o.Called()
	return args.Get(0).(ConnectionInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).(CommunicationHost), args.Error(1)
}

func (o *MockDynatraceClient) SendEvent(event *EventData) error {
	args := o.Called(event)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetEntityIDForIP(ip string) (string, error) {
	args := o.Called(ip)
	return args.String(0), args.Error(1)
}

func (o *MockDynatraceClient) GetTokenScopes(token string) (TokenScopes, error) {
	args := o.Called(token)
	return args.Get(0).(TokenScopes), args.Error(1)
}
