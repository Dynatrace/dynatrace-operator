package oneagent

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"

	testTenantUUID  = "1234"
	testTenantToken = "abcd"
	testNetworkZone = "test-zone"
)

func Test_GetConnectionInfo(t *testing.T) {
	ctx := t.Context()
	oneAgentJSONResponse := &connectionInfoJSONResponse{
		TenantUUID:                      testTenantUUID,
		TenantToken:                     testTenantToken,
		CommunicationEndpoints:          []string{testCommunicationEndpoint},
		FormattedCommunicationEndpoints: testCommunicationEndpoint,
	}
	oneAgentJSONResponseWithDups := &connectionInfoJSONResponse{
		TenantUUID:                      testTenantUUID,
		TenantToken:                     testTenantToken,
		CommunicationEndpoints:          []string{testCommunicationEndpoint, testCommunicationEndpoint},
		FormattedCommunicationEndpoints: testCommunicationEndpoint,
	}

	expectedOneAgentConnectionInfo := ConnectionInfo{
		TenantUUID:  testTenantUUID,
		TenantToken: testTenantToken,
		Endpoints:   testCommunicationEndpoint,
	}

	setupMockedClient := func(t *testing.T, params map[string]string, networkZone string, response *connectionInfoJSONResponse, err error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			WithPaasToken().
			Return(req).Once()
		req.EXPECT().
			WithQueryParams(params).
			Return(req).
			Once()
		req.EXPECT().
			Execute(&connectionInfoJSONResponse{}).
			Run(func(model any) {
				resp := model.(*connectionInfoJSONResponse)
				resp.TenantUUID = response.TenantUUID
				resp.TenantToken = response.TenantToken
				resp.FormattedCommunicationEndpoints = response.FormattedCommunicationEndpoints
				resp.CommunicationEndpoints = response.CommunicationEndpoints
			}).
			Return(err).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), connectionInfoPath).Return(req).Once()

		return NewClient(client, "", networkZone)
	}

	t.Run("no network zone", func(t *testing.T) {
		client := setupMockedClient(t, map[string]string{}, "", oneAgentJSONResponse, nil)
		connectionInfo, err := client.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})

	t.Run("with network zone", func(t *testing.T) {
		params := map[string]string{
			"networkZone":         testNetworkZone,
			"defaultZoneFallback": "true",
		}
		client := setupMockedClient(t, params, testNetworkZone, oneAgentJSONResponse, nil)
		connectionInfo, err := client.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})

	t.Run("with duplicates", func(t *testing.T) {
		client := setupMockedClient(t, map[string]string{}, "", oneAgentJSONResponseWithDups, nil)
		connectionInfo, err := client.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})

	t.Run("no communication endpoints", func(t *testing.T) {
		oneAgentJSONResponse.FormattedCommunicationEndpoints = ""
		oneAgentJSONResponse.CommunicationEndpoints = []string{}

		expectedOneAgentConnectionInfo.Endpoints = ""

		client := setupMockedClient(t, map[string]string{}, "", oneAgentJSONResponse, nil)
		connectionInfo, err := client.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedOneAgentConnectionInfo, connectionInfo)
	})

	t.Run("bad request error", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 400, Message: "bad request"}
		client := setupMockedClient(t, map[string]string{}, "", oneAgentJSONResponse, expectErr)

		_, err := client.GetConnectionInfo(ctx)
		assert.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		expectErr := errors.New("boom")
		client := setupMockedClient(t, map[string]string{}, "", oneAgentJSONResponse, expectErr)

		_, err := client.GetConnectionInfo(ctx)
		assert.ErrorIs(t, err, expectErr)
	})
}
