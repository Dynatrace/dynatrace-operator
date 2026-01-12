package dynatrace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oneAgentConnectionInfoEndpoint = "/v1/deployment/installer/agent/connectioninfo"

	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"
)

func Test_GetOneAgentConnectionInfo(t *testing.T) {
	ctx := context.Background()
	oneAgentJSONResponse := &oneAgentConnectionInfoJSONResponse{
		TenantUUID:                      testTenantUUID,
		TenantToken:                     testTenantToken,
		CommunicationEndpoints:          []string{testCommunicationEndpoint},
		FormattedCommunicationEndpoints: testCommunicationEndpoint,
	}
	oneAgentJSONResponseWithDups := &oneAgentConnectionInfoJSONResponse{
		TenantUUID:                      testTenantUUID,
		TenantToken:                     testTenantToken,
		CommunicationEndpoints:          []string{testCommunicationEndpoint, testCommunicationEndpoint},
		FormattedCommunicationEndpoints: testCommunicationEndpoint,
	}

	expectedOneAgentConnectionInfo := OneAgentConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testCommunicationEndpoint,
		},
	}

	t.Run("no network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJSONResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})

	t.Run("with duplicates", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJSONResponseWithDups), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("with network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJSONResponse), "nz")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("no communication hosts", func(t *testing.T) {
		oneAgentJSONResponse.FormattedCommunicationEndpoints = ""
		oneAgentJSONResponse.CommunicationEndpoints = []string{}

		expectedOneAgentConnectionInfo.Endpoints = ""

		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJSONResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("with non-existent network zone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentConnectionInfoEndpoint, oneAgentJSONResponse), "")
		defer dynatraceServer.Close()

		connectionInfo, err := dynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})
	t.Run("handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJSON(oneAgentConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.Error(t, err)
		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())

		assert.NotNil(t, connectionInfo)
		assert.Equal(t, OneAgentConnectionInfo{}, connectionInfo)
	})
	t.Run("handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(oneAgentConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		connectionInfo, err := faultyDynatraceClient.GetOneAgentConnectionInfo(ctx)
		require.Error(t, err)
		assert.NotNil(t, connectionInfo)
		assert.Equal(t, OneAgentConnectionInfo{}, connectionInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}
