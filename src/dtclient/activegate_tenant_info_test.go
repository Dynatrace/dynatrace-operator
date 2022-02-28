package dtclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const activeGateConnectionInfoEndpoint = "/v1/deployment/installer/gateway/connectioninfo"

var agTenantResponse = &ActiveGateTenantInfo{
	TenantInfo{"abcd", "1234"},
	"/some/url",
}

func TestGetActiveGateTenantInfoFaulty1(t *testing.T) {
	t.Run("GetActiveGateTenantInfo with no networkzone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(activeGateConnectionInfoEndpoint, agTenantResponse), "")
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetActiveGateTenantInfo()
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, agTenantResponse, tenantInfo)
	})
	t.Run("GetActiveGateTenantInfo with networkzone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(activeGateConnectionInfoEndpoint, agTenantResponse), "nz")
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetActiveGateTenantInfo()
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, agTenantResponse, tenantInfo)
	})
	t.Run("GetActiveGateTenantInfo with nonexisting networkzone", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(activeGateConnectionInfoEndpoint, agTenantResponse), "")
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetActiveGateTenantInfo()
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, agTenantResponse, tenantInfo)
	})
	t.Run("GetActiveGateTenantInfo handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("GetActiveGateTenantInfo handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError(activeGateConnectionInfoEndpoint), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}
