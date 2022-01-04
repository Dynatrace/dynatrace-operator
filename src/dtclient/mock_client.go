package dtclient

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockDynatraceClient implements a Dynatrace REST API Client mock
type MockDynatraceClient struct {
	mock.Mock
}

func (o *MockDynatraceClient) GetTenantInfo() (*TenantInfo, error) {
	args := o.Called()
	return args.Get(0).(*TenantInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgent(os, installerType, flavor, arch string, writer io.Writer) error {
	args := o.Called(os, installerType, flavor, arch, writer)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetAgent(os, installerType, flavor, arch, version string, writer io.Writer) error {
	args := o.Called(os, installerType, flavor, arch, version, writer)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetAgentVersions(os, installerType, flavor, arch string) ([]string, error) {
	args := o.Called(os, installerType, flavor, arch)
	return args.Get(0).([]string), args.Error(1)
}

func (o *MockDynatraceClient) GetConnectionInfo() (ConnectionInfo, error) {
	args := o.Called()
	return args.Get(0).(ConnectionInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).(CommunicationHost), args.Error(1)
}

func (o *MockDynatraceClient) GetProcessModuleConfig(prevRevision uint) (*ProcessModuleConfig, error) {
	args := o.Called(prevRevision)
	return args.Get(0).(*ProcessModuleConfig), args.Error(1)
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

func (o *MockDynatraceClient) CreateSetting(label string, kubeSystemUUID string) (string, error) {
	args := o.Called(label, kubeSystemUUID)
	return args.String(0), args.Error(1)
}
