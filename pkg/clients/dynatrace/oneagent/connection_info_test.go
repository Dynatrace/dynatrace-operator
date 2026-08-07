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
		require.ErrorIs(t, err, NoCommunicationEndpointsError)
		assert.Equal(t, expected, connectionInfo)
	})

	t.Run("required IPs missing from response → StaleNetworkZoneEndpointsError", func(t *testing.T) {
		oaClient := setupMockedClient(t, map[string]string{}, "", response, nil)
		_, err := oaClient.GetConnectionInfo(ctx, []string{"192.0.2.1"})
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

func Test_buildEndpoints(t *testing.T) {
	const (
		epA              = "https://tenant.dev.dynatracelabs.com:443"
		epB              = "https://other.dev.dynatracelabs.com:8443"
		epC              = "https://third.dev.dynatracelabs.com:443"
		localServiceHost = "test-dk-activegate.dynatrace"
	)

	t.Run("deduplication", func(t *testing.T) {
		tests := []struct {
			name      string
			input     []string
			expected  string
			expectErr error
		}{
			{name: "nil slice", input: nil, expected: "", expectErr: NoCommunicationEndpointsError},
			{name: "empty slice", input: []string{}, expected: "", expectErr: NoCommunicationEndpointsError},
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
				got, err := buildEndpoints(tt.input, nil)
				assert.Equal(t, tt.expected, got)
				assert.ErrorIs(t, err, tt.expectErr)
			})
		}
	})

	t.Run("missing IPs", func(t *testing.T) {
		agEP := "https://" + localServiceHost + ":443/communication"

		tests := []struct {
			name        string
			endpoints   []string
			requiredIPs []string
			expect      error
		}{
			{
				name:        "no required IPs → not missing",
				endpoints:   []string{"anything"},
				requiredIPs: nil,
				expect:      nil,
			},
			{
				name:        "empty required IPs → not missing",
				endpoints:   []string{"anything"},
				requiredIPs: []string{},
				expect:      nil,
			},
			{
				name:        "required IP present alongside local AG DNS endpoint → not missing",
				endpoints:   []string{"https://10.0.0.1:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1"},
				expect:      nil,
			},
			{
				name:        "required IP present alongside unrelated endpoints → not missing",
				endpoints:   []string{"https://1.2.3.4:443/communication", "https://10.0.0.1:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1"},
				expect:      nil,
			},
			{
				name:        "IPv6 required IP present (bracketed in endpoint URL) → not missing",
				endpoints:   []string{"https://[2001:db8::1]:443/communication", agEP},
				requiredIPs: []string{"2001:db8::1"},
				expect:      nil,
			},
			{
				name:        "required IP missing from endpoints → missing",
				endpoints:   []string{"https://10.0.0.1:443/communication", agEP},
				requiredIPs: []string{"10.0.0.2"},
				expect:      StaleNetworkZoneEndpointsError,
			},
			{
				name:        "empty endpoints with required IPs → missing",
				endpoints:   []string{},
				requiredIPs: []string{"10.0.0.1"},
				expect:      NoCommunicationEndpointsError,
			},
			{
				name:        "endpoints contain no IP-based entries at all → missing",
				endpoints:   []string{"https://other-activegate.dynatrace:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1"},
				expect:      StaleNetworkZoneEndpointsError,
			},
			{
				name:        "dual-stack: all required IPs present → not missing",
				endpoints:   []string{"https://10.0.0.1:443/communication", "https://[2001:db8::1]:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1", "2001:db8::1"},
				expect:      nil,
			},
			{
				name:        "dual-stack: one required IP missing → not missing",
				endpoints:   []string{"https://10.0.0.1:443/communication", "https://[2001:db8::2]:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1", "2001:db8::1"},
				expect:      nil,
			},
			{
				name:        "unparseable entry is skipped, required IP still found → not missing",
				endpoints:   []string{"garbage", "https://10.0.0.1:443/communication", agEP},
				requiredIPs: []string{"10.0.0.1"},
				expect:      nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := buildEndpoints(tt.endpoints, tt.requiredIPs)
				assert.ErrorIs(t, err, tt.expect)
			})
		}
	})
}
