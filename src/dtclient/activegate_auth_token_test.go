package dtclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const activeGateAuthTokenUrl = "/v2/activeGateTokens"
const dynakubeName = "dynakube"

var activeGateAuthTokenResponse = &ActiveGateAuthTokenInfo{
	TokenId: "test",
	Token:   "dt.some.valuegoeshere",
}

func TestGetActiveGateAuthTokenInfo(t *testing.T) {
	t.Run("GetActiveGateAuthToken works", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceClient(t, tenantServerHandler(activeGateAuthTokenUrl, activeGateAuthTokenResponse), "")
		defer dynatraceServer.Close()

		agAuthTokenInfo, err := dynatraceClient.GetActiveGateAuthToken(dynakubeName)
		assert.NoError(t, err)
		assert.NotNil(t, agAuthTokenInfo)

		assert.Equal(t, activeGateAuthTokenResponse, agAuthTokenInfo)
	})
	t.Run("GetActiveGateAuthToken handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantMalformedJson(activeGateAuthTokenUrl), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateAuthToken(dynakubeName)
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("GetActiveGateAuthToken handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceClient(t, tenantInternalServerError(activeGateAuthTokenUrl), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateAuthToken(dynakubeName)
		assert.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}
