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
			Execute(&connectionInfoResponse{requiredHosts: response.requiredHosts}).
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
		require.ErrorIs(t, err, NoCommunicationEndpointsError)
		assert.Equal(t, expected, connectionInfo)
	})

	t.Run("required IPs missing from response → StaleNetworkZoneEndpointsError", func(t *testing.T) {
		requiredHosts := []string{"192.0.2.1"}
		response := &connectionInfoResponse{
			TenantUUID:             testTenantUUID,
			TenantToken:            testTenantToken,
			CommunicationEndpoints: []string{testCommunicationEndpoint},
			requiredHosts:          requiredHosts,
		}
		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		_, err := oaClient.GetConnectionInfo(ctx, requiredHosts)
		require.Error(t, err)
		assert.ErrorIs(t, err, StaleNetworkZoneEndpointsError)
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

func Test_connectionInfoResponse_IsEmpty(t *testing.T) {
	const (
		validUUID     = "uuid"
		validToken    = "token"
		validEndpoint = "https://10.0.0.1:443/communication"
		validHost     = "10.0.0.1"
		otherEndpoint = "https://10.0.0.2:443/communication"
		otherHost     = "10.0.0.2"
	)

	tests := []struct {
		name     string
		response *connectionInfoResponse
		want     bool
	}{
		{
			name: "all fields set, no required hosts",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				TenantToken:            validToken,
				CommunicationEndpoints: []string{validEndpoint},
			},
			want: false,
		},
		{
			name: "missing TenantUUID",
			response: &connectionInfoResponse{
				TenantToken:            validToken,
				CommunicationEndpoints: []string{validEndpoint},
			},
			want: true,
		},
		{
			name: "missing TenantToken",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				CommunicationEndpoints: []string{validEndpoint},
			},
			want: true,
		},
		{
			name: "no communication endpoints",
			response: &connectionInfoResponse{
				TenantUUID:  validUUID,
				TenantToken: validToken,
			},
			want: true,
		},
		{
			name: "required host present in endpoints",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				TenantToken:            validToken,
				CommunicationEndpoints: []string{validEndpoint},
				requiredHosts:          []string{validHost},
			},
			want: false,
		},
		{
			name: "required host absent from endpoints",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				TenantToken:            validToken,
				CommunicationEndpoints: []string{validEndpoint},
				requiredHosts:          []string{otherHost},
			},
			want: true,
		},
		{
			name: "required host found in one of multiple endpoints",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				TenantToken:            validToken,
				CommunicationEndpoints: []string{validEndpoint, otherEndpoint},
				requiredHosts:          []string{otherHost},
			},
			want: false,
		},
		{
			name: "unparseable endpoint with required host counts as missing",
			response: &connectionInfoResponse{
				TenantUUID:             validUUID,
				TenantToken:            validToken,
				CommunicationEndpoints: []string{"not-a-url"},
				requiredHosts:          []string{validHost},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.response.IsEmpty())
		})
	}
}

func Test_connectionInfoResponse_uniqueEndpoints(t *testing.T) {
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
		{name: "nil slice", input: nil, expected: ""},
		{name: "empty slice", input: []string{}, expected: ""},
		{name: "single endpoint", input: []string{epA}, expected: epA},
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
		{name: "all duplicates collapse to one", input: []string{epA, epA, epA}, expected: epA},
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
			r := &connectionInfoResponse{CommunicationEndpoints: tt.input}
			got := strings.Join(r.uniqueEndpoints(), ";")
			assert.Equal(t, tt.expected, got)
		})
	}
}
