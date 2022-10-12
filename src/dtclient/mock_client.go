package dtclient

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockDynatraceClient implements a Dynatrace REST API Client mock
type MockDynatraceClient struct {
	mock.Mock
}

func (o *MockDynatraceClient) GetActiveGateConnectionInfo() (*ActiveGateConnectionInfo, error) {
	args := o.Called()
	return args.Get(0).(*ActiveGateConnectionInfo), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

func (o *MockDynatraceClient) GetLatestAgent(os, installerType, flavor, arch string, technologies []string, writer io.Writer) error {
	args := o.Called(os, installerType, flavor, arch, technologies, writer)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetAgent(os, installerType, flavor, arch, version string, technologies []string, writer io.Writer) error {
	args := o.Called(os, installerType, flavor, arch, version, technologies, writer)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetAgentViaInstallerUrl(url string, writer io.Writer) error {
	args := o.Called(url, writer)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetAgentVersions(os, installerType, flavor, arch string) ([]string, error) {
	args := o.Called(os, installerType, flavor, arch)
	return args.Get(0).([]string), args.Error(1)
}

func (o *MockDynatraceClient) GetOneAgentConnectionInfo() (OneAgentConnectionInfo, error) {
	args := o.Called()
	return args.Get(0).(OneAgentConnectionInfo), args.Error(1)
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

func (o *MockDynatraceClient) CreateOrUpdateKubernetesSetting(name string, kubeSystemUUID string, scope string) (string, error) {
	args := o.Called(name, kubeSystemUUID, scope)
	return args.String(0), args.Error(1)
}

func (o *MockDynatraceClient) GetMonitoredEntitiesForKubeSystemUUID(kubeSystemUUID string) ([]MonitoredEntity, error) {
	args := o.Called(kubeSystemUUID)
	return args.Get(0).([]MonitoredEntity), args.Error(1)
}

func (o *MockDynatraceClient) GetSettingsForMonitoredEntities(monitoredEntities []MonitoredEntity) (GetSettingsResponse, error) {
	args := o.Called(monitoredEntities)
	return args.Get(0).(GetSettingsResponse), args.Error(1)
}

func (o *MockDynatraceClient) GetActiveGateAuthToken(dynakubeName string) (*ActiveGateAuthTokenInfo, error) {
	args := o.Called(dynakubeName)
	return args.Get(0).(*ActiveGateAuthTokenInfo), args.Error(1)
}
