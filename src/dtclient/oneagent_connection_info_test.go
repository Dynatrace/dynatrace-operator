package dtclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	oneAgentConnectionInfoEndpoint = "/v1/deployment/installer/agent/connectioninfo"

	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"
)

func Test_GetOneAgentConnectionInfo(t *testing.T) {
	oneAgentJsonResponse := &oneAgentConnectionInfoJsonResponse{
		TenantUUID:                      testTenantUUID,
		TenantToken:                     testTenantToken,
		CommunicationEndpoints:          []string{testCommunicationEndpoint},
		FormattedCommunicationEndpoints: testCommunicationEndpoint,
	}

	expectedOneAgentConnectionInfo := OneAgentConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testCommunicationEndpoint,
		},
		CommunicationHosts: []CommunicationHost{
			{
				Protocol: "https",
				Host:     "tenant.dev.dynatracelabs.com",
				Port:     443,
			},
		},
	}

	t.Run("no network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJsonResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("with network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJsonResponse), "nz")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("no communication hosts", func(t *testing.T) {
		oneAgentJsonResponse.FormattedCommunicationEndpoints = ""
		oneAgentJsonResponse.CommunicationEndpoints = []string{}

		expectedOneAgentConnectionInfo.CommunicationHosts = []CommunicationHost{}
		expectedOneAgentConnectionInfo.Endpoints = ""

		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJsonResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("with non-existent network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJsonResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo()
		assert.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJson(oneAgentConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetOneAgentConnectionInfo()
		assert.Error(t, err)
		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())

		assert.NotNil(t, connectionInfo)
		assert.Equal(t, OneAgentConnectionInfo{}, connectionInfo)
	})
	t.Run("handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(oneAgentConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetOneAgentConnectionInfo()
		assert.Error(t, err)
		assert.NotNil(t, connectionInfo)
		assert.Equal(t, OneAgentConnectionInfo{}, connectionInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}
