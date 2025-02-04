package dynatrace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	activeGateAuthTokenUrl = "/v2/activeGateTokens"
	dynakubeName           = "dynakube"
)

var activeGateAuthTokenResponse = &ActiveGateAuthTokenInfo{
	TokenId: "test",
	Token:   "dt.some.valuegoeshere",
}

func TestGetActiveGateAuthTokenInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("GetActiveGateAuthToken works", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(activeGateAuthTokenUrl, activeGateAuthTokenResponse), "")
		defer dynatraceServer.Close()

		agAuthTokenInfo, err := dynatraceClient.GetActiveGateAuthToken(ctx, dynakubeName)
		require.NoError(t, err)
		assert.NotNil(t, agAuthTokenInfo)

		assert.Equal(t, activeGateAuthTokenResponse, agAuthTokenInfo)
	})
	t.Run("GetActiveGateAuthToken handle malformed json", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJson(activeGateAuthTokenUrl), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateAuthToken(ctx, dynakubeName)
		require.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("GetActiveGateAuthToken handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(activeGateAuthTokenUrl), "")
		defer faultyDynatraceServer.Close()

		tenantInfo, err := faultyDynatraceClient.GetActiveGateAuthToken(ctx, dynakubeName)
		require.Error(t, err)
		assert.Nil(t, tenantInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
}
