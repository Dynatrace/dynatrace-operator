// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"errors"
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
	response := &connectionInfoResponse{
		TenantUUID:             testTenantUUID,
		TenantToken:            testTenantToken,
		CommunicationEndpoints: []string{testCommunicationEndpoint},
	}

	expectedResponse := ConnectionInfo{
		TenantUUID:  testTenantUUID,
		TenantToken: testTenantToken,
		Endpoints:   testCommunicationEndpoint,
	}

	setupMockedClient := func(t *testing.T, params map[string]string, networkZone string, response *connectionInfoResponse, err error) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().WithQueryParams(params).Return(req).Once()
		req.EXPECT().
			Execute(&connectionInfoResponse{}).
			Run(func(model any) {
				resp := model.(*connectionInfoResponse)
				resp.TenantUUID = response.TenantUUID
				resp.TenantToken = response.TenantToken
				resp.CommunicationEndpoints = response.CommunicationEndpoints
			}).
			Return(err).Once()
		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(anyCtx, connectionInfoPath).Return(req).Once()

		return NewClient(coreClient, "", networkZone)
	}

	t.Run("no network zone", func(t *testing.T) {
		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx, nil)
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
		connectionInfo, err := oaClient.GetConnectionInfo(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expectedResponse, connectionInfo)
	})

	t.Run("endpoints are derived from the deduplicated communicationEndpoints slice", func(t *testing.T) {
		dupResponse := &connectionInfoResponse{
			TenantUUID:             testTenantUUID,
			TenantToken:            testTenantToken,
			CommunicationEndpoints: []string{testCommunicationEndpoint, testCommunicationEndpoint},
		}
		expected := ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testCommunicationEndpoint,
		}

		oaClient := setupMockedClient(t, map[string]string{}, "", dupResponse, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx, nil)
		require.NoError(t, err)

		assert.Equal(t, expected, connectionInfo)
	})

	t.Run("no communication endpoints", func(t *testing.T) {
		emptyResponse := &connectionInfoResponse{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
		}
		expected := ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   "",
		}

		oaClient := setupMockedClient(t, map[string]string{}, "", emptyResponse, nil)
		connectionInfo, err := oaClient.GetConnectionInfo(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, connectionInfo)

		assert.Equal(t, expected, connectionInfo)
	})

	t.Run("required IPS missing from response → MissingIPs error", func(t *testing.T) {
		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		_, err := oaClient.GetConnectionInfo(ctx, []string{"192.0.2.1"})
		require.Error(t, err)
		assert.ErrorAs(t, err, new(MissingIPs))
	})

	t.Run("bad request error", func(t *testing.T) {
		expectErr := &core.HTTPError{StatusCode: 400, Message: "bad request"}
		oaClient := setupMockedClient(t, map[string]string{}, "", response, expectErr)

		_, err := oaClient.GetConnectionInfo(ctx, nil)
		assert.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		expectErr := errors.New("boom")
		oaClient := setupMockedClient(t, map[string]string{}, "", response, expectErr)

		_, err := oaClient.GetConnectionInfo(ctx, nil)
		assert.ErrorIs(t, err, expectErr)
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

func Test_missingEndpoint(t *testing.T) {
	const (
		localServiceHost = "test-dk-activegate.dynatrace"
	)

	t.Run("no required IPs → not missing", func(t *testing.T) {
		assert.False(t, missingIPs("anything", nil))
	})

	t.Run("empty required IPs → not missing", func(t *testing.T) {
		assert.False(t, missingIPs("anything", []string{}))
	})

	t.Run("required IP present alongside local AG DNS endpoint → not missing", func(t *testing.T) {
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, missingIPs(endpoints, []string{"10.0.0.1"}))
	})

	t.Run("required IP present alongside unrelated endpoints → not missing", func(t *testing.T) {
		endpoints := "https://1.2.3.4:443/communication;https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, missingIPs(endpoints, []string{"10.0.0.1"}))
	})

	t.Run("IPv6 required IP present (bracketed in endpoint URL) → not missing", func(t *testing.T) {
		endpoints := "https://[2001:db8::1]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, missingIPs(endpoints, []string{"2001:db8::1"}))
	})

	t.Run("required IP missing from endpoints → missing", func(t *testing.T) {
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, missingIPs(endpoints, []string{"10.0.0.2"}))
	})

	t.Run("empty endpoints with required IPs → missing", func(t *testing.T) {
		assert.True(t, missingIPs("", []string{"10.0.0.1"}))
	})

	t.Run("endpoints contain no IP-based entries at all → missing", func(t *testing.T) {
		endpoints := "https://other-activegate.dynatrace:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, missingIPs(endpoints, []string{"10.0.0.1"}))
	})

	t.Run("dual-stack: all required IPs present → not missing", func(t *testing.T) {
		endpoints := "https://10.0.0.1:443/communication;https://[2001:db8::1]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, missingIPs(endpoints, []string{"10.0.0.1", "2001:db8::1"}))
	})

	t.Run("dual-stack: one required IP missing → missing", func(t *testing.T) {
		endpoints := "https://10.0.0.1:443/communication;https://[2001:db8::2]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, missingIPs(endpoints, []string{"10.0.0.1", "2001:db8::1"}))
	})

	t.Run("unparseable endpoint string → not missing (best effort skip)", func(t *testing.T) {
		endpoints := "garbage;https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, missingIPs(endpoints, []string{"10.0.0.1"}))
	})
}
