package dtclient

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const activeGateConnectionInfoEndpoint = "/v1/deployment/installer/gateway/connectioninfo"

var agTenantResponse = ActiveGateTenantInfo{"abcd", "1234", "/some/url"}

func TestGetActiveGateTenantInfo(t *testing.T) {
	t.Run("GetActiveGateTenantInfo", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(activeGateConnectionInfoEndpoint, agTenantResponse))
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetActiveGateTenantInfo(false)
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, agTenantResponse.UUID, tenantInfo.UUID)
		assert.Equal(t, agTenantResponse.Token, tenantInfo.Token)
		assert.Equal(t, agTenantResponse.Endpoints, tenantInfo.Endpoints)
	})
	t.Run("GetActiveGateTenantInfo handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError(activeGateConnectionInfoEndpoint))
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateTenantInfo(false)
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetActiveGateTenantInfo handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson(activeGateConnectionInfoEndpoint))
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateTenantInfo(false)
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
}
