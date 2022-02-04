package dtclient

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const agentConnectionInfoEndpoint = "/v1/deployment/installer/agent/connectioninfo"

var agentTenantResponse = struct {
	TenantUUID             string
	TenantToken            string
	CommunicationEndpoints []string
}{
	TenantUUID:             "abcd",
	TenantToken:            "1234",
	CommunicationEndpoints: []string{"/some/url"},
}

func TestTenant(t *testing.T) {
	t.Run("GetAgentTenantInfo", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(agentConnectionInfoEndpoint, agentTenantResponse))
		defer dynatraceServer.Close()

		tenantInfo, err := dynatraceClient.GetAgentTenantInfo()
		assert.NoError(t, err)
		assert.NotNil(t, tenantInfo)

		assert.Equal(t, agentTenantResponse.TenantUUID, tenantInfo.UUID)
		assert.Equal(t, agentTenantResponse.TenantToken, tenantInfo.Token)
		assert.Equal(t, agentTenantResponse.CommunicationEndpoints, tenantInfo.Endpoints)
		assert.Equal(t,
			strings.Join([]string{agentTenantResponse.CommunicationEndpoints[0], DtCommunicationSuffix}, Slash),
			tenantInfo.CommunicationEndpoint)
	})
	t.Run("GetAgentTenantInfo handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError(agentConnectionInfoEndpoint))
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetAgentTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetAgentTenantInfo handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson(agentConnectionInfoEndpoint))
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetAgentTenantInfo()
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
}
