// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"errors"
	"strings"
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
		TenantUUID:  testTenantUUID,
		TenantToken: testTenantToken,
		Endpoints:   testCommunicationEndpoint,
	}

	expectedResponse := ConnectionInfo{
		TenantUUID:  testTenantUUID,
		TenantToken: testTenantToken,
		Endpoints:   testCommunicationEndpoint,
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

	t.Run("no communication endpoints", func(t *testing.T) {
		response.Endpoints = ""
		expectedResponse.Endpoints = ""

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
		response.Endpoints = testCommunicationEndpoint + ";" + testCommunicationEndpoint
		expectedResponse.Endpoints = testCommunicationEndpoint

		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx)
		require.NoError(t, err)

		assert.Equal(t, expectedResponse, connectionInfo)
	})
}

func Test_deduplicateEndpoints(t *testing.T) {
	const (
		epA = "https://tenant.dev.dynatracelabs.com:443"
		epB = "https://other.dev.dynatracelabs.com:8443"
		epC = "https://third.dev.dynatracelabs.com:443"
	)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single endpoint",
			input:    epA,
			expected: epA,
		},
		{
			name:     "no duplicates is a no-op",
			input:    epA + ";" + epB + ";" + epC,
			expected: epA + ";" + epB + ";" + epC,
		},
		{
			name:     "some duplicates preserve first-occurrence order",
			input:    epB + ";" + epA + ";" + epB + ";" + epC + ";" + epA,
			expected: epB + ";" + epA + ";" + epC,
		},
		{
			name:     "all duplicates collapse to one",
			input:    epA + ";" + epA + ";" + epA,
			expected: epA,
		},
		{
			name:     "endpoints differing only by surrounding whitespace are duplicates",
			input:    epA + "; " + epA + " ;\t" + epA,
			expected: epA,
		},
		{
			name:     "empty segments are dropped",
			input:    epA + ";;" + epB + ";",
			expected: epA + ";" + epB,
		},
		{
			name:     "endpoints differing only by case are kept as distinct",
			input:    epA + ";" + strings.ToUpper(epA),
			expected: epA + ";" + strings.ToUpper(epA),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, deduplicateEndpoints(tt.input))
		})
	}
}
