// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

const (
	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"

	testTenantUUID  = "1234"
	testTenantToken = "abcd"
	testNetworkZone = "test-zone"
)

func Test_GetConnectionInfo(t *testing.T) {
	ctx := t.Context()
	response := &ConnectionInfo{
		TenantUUID:             testTenantUUID,
		TenantToken:            testTenantToken,
		CommunicationEndpoints: []string{testCommunicationEndpoint},
	}

	expectedResponse := ConnectionInfo{
		TenantUUID:             testTenantUUID,
		TenantToken:            testTenantToken,
		Endpoints:              testCommunicationEndpoint,
		CommunicationEndpoints: []string{testCommunicationEndpoint},
	}

	setupMockedClient := func(t *testing.T, params map[string]string, networkZone string, response *ConnectionInfo, err error) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().WithQueryParams(params).Return(req).Once()
		req.EXPECT().
			Execute(&ConnectionInfo{}).
			Run(func(model any) {
				resp := model.(*ConnectionInfo)
				resp.TenantUUID = response.TenantUUID
				resp.TenantToken = response.TenantToken
				resp.Endpoints = response.Endpoints
				resp.CommunicationEndpoints = response.CommunicationEndpoints
			}).
			Return(err).Once()
		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(anyCtx, connectionInfoPath).Return(req).Once()

		return NewClient(coreClient, "", networkZone)
	}

	t.Run("no network zone", func(t *testing.T) {
		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedResponse, connectionInfo)
	})

	t.Run("with network zone", func(t *testing.T) {
		params := map[string]string{
			"networkZone":         testNetworkZone,
			"defaultZoneFallback": "true",
		}
		oaClient := setupMockedClient(t, params, testNetworkZone, response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedResponse, connectionInfo)
	})

	t.Run("endpoints are derived from the deduplicated communicationEndpoints slice", func(t *testing.T) {
		dupResponse := &ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			// The formatted string coming from the API is ignored; the slice wins.
			Endpoints:              "https://stale.example.com:443",
			CommunicationEndpoints: []string{testCommunicationEndpoint, testCommunicationEndpoint},
		}
		expected := ConnectionInfo{
			TenantUUID:             testTenantUUID,
			TenantToken:            testTenantToken,
			Endpoints:              testCommunicationEndpoint,
			CommunicationEndpoints: []string{testCommunicationEndpoint, testCommunicationEndpoint},
		}

		oaClient := setupMockedClient(t, map[string]string{}, "", dupResponse, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)

		assert.Equal(t, expected, connectionInfo)
	})

	t.Run("no communication endpoints", func(t *testing.T) {
		response.Endpoints = ""
		response.CommunicationEndpoints = nil
		expectedResponse.Endpoints = ""
		expectedResponse.CommunicationEndpoints = nil

		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedResponse, connectionInfo)
	})

	t.Run("bad request error", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 400, Message: "bad request"}
		oaClient := setupMockedClient(t, map[string]string{}, "", response, expectErr)

		_, err := oaClient.GetConnectionInfo(ctx)
		assert.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		expectErr := errors.New("boom")
		oaClient := setupMockedClient(t, map[string]string{}, "", response, expectErr)

		_, err := oaClient.GetConnectionInfo(ctx)
		assert.ErrorIs(t, err, expectErr)
	})

	t.Run("duplicate endpoints are deduplicated", func(t *testing.T) {
		response.CommunicationEndpoints = []string{testCommunicationEndpoint, testCommunicationEndpoint}
		expectedResponse.Endpoints = testCommunicationEndpoint
		expectedResponse.CommunicationEndpoints = []string{testCommunicationEndpoint, testCommunicationEndpoint}

		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)

		assert.Equal(t, expectedResponse, connectionInfo)
	})
}

// Test_GetConnectionInfo_EndToEnd exercises the full path through the real core
// HTTP client against an httptest.Server. The mocked JSON response contains both
// the formatted string and a communicationEndpoints array with duplicates, and we
// assert the returned Endpoints string is derived from the slice with duplicates
// removed and the entries joined by endpointDelimiter.
func Test_GetConnectionInfo_EndToEnd(t *testing.T) {
	const (
		epA = "https://tenant.dev.dynatracelabs.com:443"
		epB = "https://other.dev.dynatracelabs.com:8443"
	)

	body := `{
		"tenantUUID": "` + testTenantUUID + `",
		"tenantToken": "` + testTenantToken + `",
		"formattedCommunicationEndpoints": "https://stale.example.com:443",
		"communicationEndpoints": ["` + epA + `", "` + epB + `", "` + epA + `"]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, connectionInfoPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	coreClient := core.NewClient(core.Config{
		BaseURL:   mustParseURL(t, server.URL),
		PaasToken: "paas",
	})
	oaClient := NewClient(coreClient, "", "")

	connectionInfo, err := oaClient.GetConnectionInfo(t.Context())
	require.NoError(t, err)

	assert.Equal(t, testTenantUUID, connectionInfo.TenantUUID)
	assert.Equal(t, testTenantToken, connectionInfo.TenantToken)
	assert.Equal(t, epA+";"+epB, connectionInfo.Endpoints)
}

func Test_deduplicateEndpoints(t *testing.T) {
	const (
		epA = "https://tenant.dev.dynatracelabs.com:443"
		epB = "https://other.dev.dynatracelabs.com:8443"
		epC = "https://third.dev.dynatracelabs.com:443"
	)

	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single endpoint",
			input:    []string{epA},
			expected: epA,
		},
		{
			name:     "no duplicates is a no-op, order preserved",
			input:    []string{epA, epB, epC},
			expected: epA + ";" + epB + ";" + epC,
		},
		{
			name:     "some duplicates preserve first-occurrence order",
			input:    []string{epB, epA, epB, epC, epA},
			expected: epB + ";" + epA + ";" + epC,
		},
		{
			name:     "all duplicates collapse to one",
			input:    []string{epA, epA, epA},
			expected: epA,
		},
		{
			name:     "entries differing only by surrounding whitespace are trimmed and deduplicated",
			input:    []string{epA, " " + epA, epA + "\t"},
			expected: epA,
		},
		{
			name:     "empty and whitespace-only entries are dropped",
			input:    []string{epA, "", "   ", epB},
			expected: epA + ";" + epB,
		},
		{
			name:     "entries differing only by case are kept as distinct",
			input:    []string{epA, "HTTPS://TENANT.DEV.DYNATRACELABS.COM:443"},
			expected: epA + ";" + "HTTPS://TENANT.DEV.DYNATRACELABS.COM:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, deduplicateEndpoints(tt.input))
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	require.NoError(t, err)

	return u
}
