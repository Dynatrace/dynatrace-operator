// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oaconnectioninfo

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConnectionInfoCacheTTL = time.Minute

func TestReconcile_ConnectionInfoCache(t *testing.T) {
	t.Run("uses cached connection info within TTL", func(t *testing.T) {
		transport := newFakeCITransport(
			connectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
		)
		oaClient := newConnectionInfoOAClient(t, transport, testConnectionInfoCacheTTL)
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, fakeClient)

		require.NoError(t, r.Reconcile(t.Context(), oaClient, dk))
		require.NoError(t, r.Reconcile(t.Context(), oaClient, dk))

		transport.assertCalls(t, 1, "second reconcile must be served from cache")
	})

	t.Run("fetches fresh connection info after cache TTL expires", func(t *testing.T) {
		const updatedUUID = "updated-uuid"
		transport := newFakeCITransport(
			connectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
			connectionInfoBody(updatedUUID, testTenantToken, testTenantEndpoints),
		)
		oaClient := newConnectionInfoOAClient(t, transport, testConnectionInfoCacheTTL)
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, fakeClient)

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, r.Reconcile(t.Context(), oaClient, dk))

			time.Sleep(testConnectionInfoCacheTTL + time.Second)

			require.NoError(t, r.Reconcile(t.Context(), oaClient, dk))
		})

		assert.Equal(t, updatedUUID, dk.Status.OneAgent.ConnectionInfo.TenantUUID)
		transport.assertCalls(t, 2)
	})

	t.Run("invalidates cache when enpoints are absent from response", func(t *testing.T) {
		transport := newFakeCITransport(
			connectionInfoEmptyEndpointsBody(testTenantUUID, testTenantToken),
			connectionInfoEmptyEndpointsBody(testTenantUUID, testTenantToken),
		)
		oaClient := newConnectionInfoOAClient(t, transport, testConnectionInfoCacheTTL)

		_, err := oaClient.GetConnectionInfo(t.Context(), nil)
		require.ErrorIs(t, err, oneagentclient.NoCommunicationEndpointsError)

		_, err = oaClient.GetConnectionInfo(t.Context(), nil)
		require.ErrorIs(t, err, oneagentclient.NoCommunicationEndpointsError)

		transport.assertCalls(t, 2, "cache must be invalidated on each call when endpoints absent")
	})

	t.Run("invalidates cache when required host is absent from response", func(t *testing.T) {
		const (
			knownHost     = "10.0.0.1"
			knownEndpoint = "https://" + knownHost + ":443/communication"
			requiredHost  = "10.0.0.2"
		)
		transport := newFakeCITransport(
			connectionInfoBody(testTenantUUID, testTenantToken, knownEndpoint),
			connectionInfoBody(testTenantUUID, testTenantToken, knownEndpoint),
		)
		oaClient := newConnectionInfoOAClient(t, transport, testConnectionInfoCacheTTL)

		_, err := oaClient.GetConnectionInfo(t.Context(), []string{requiredHost})
		require.ErrorIs(t, err, oneagentclient.StaleNetworkZoneEndpointsError)

		_, err = oaClient.GetConnectionInfo(t.Context(), []string{requiredHost})
		require.ErrorIs(t, err, oneagentclient.StaleNetworkZoneEndpointsError)

		transport.assertCalls(t, 2, "cache must be invalidated on each call when required host is absent")
	})

	t.Run("keeps cache when required host is present in response", func(t *testing.T) {
		const (
			knownHost     = "10.0.0.1"
			knownEndpoint = "https://" + knownHost + ":443/communication"
		)
		transport := newFakeCITransport(
			connectionInfoBody(testTenantUUID, testTenantToken, knownEndpoint),
		)
		oaClient := newConnectionInfoOAClient(t, transport, testConnectionInfoCacheTTL)

		_, err := oaClient.GetConnectionInfo(t.Context(), []string{knownHost})
		require.NoError(t, err)

		_, err = oaClient.GetConnectionInfo(t.Context(), []string{knownHost})
		require.NoError(t, err)

		transport.assertCalls(t, 1, "cache must be kept when required host is present")
	})
}

type fakeCITransport struct {
	calls  atomic.Int64
	bodies []string
}

func newFakeCITransport(bodies ...string) *fakeCITransport {
	return &fakeCITransport{bodies: bodies}
}

func (ft *fakeCITransport) assertCalls(t *testing.T, expected int64, msgAndArgs ...any) {
	t.Helper()
	assert.Equal(t, expected, ft.calls.Load(), msgAndArgs...)
}

func (ft *fakeCITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	idx := int(ft.calls.Add(1)) - 1
	body := ft.bodies[min(idx, len(ft.bodies)-1)]

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

// newConnectionInfoOAClient builds a real oneagent.ClientImpl whose HTTP transport
// is the given fakeCITransport wrapped in a cache round-tripper with the specified TTL.
// PaasToken is set to t.Name() so each subtest has an isolated namespace in the global cache.
func newConnectionInfoOAClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) oneagentclient.Client {
	t.Helper()

	u, err := url.Parse("https://fake-dt.test")
	require.NoError(t, err)

	return oneagentclient.NewClient(core.NewClient(core.Config{
		BaseURL:    u,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		PaasToken:  t.Name(),
	}), "", "")
}

func connectionInfoBody(tenantUUID, tenantToken, endpoint string) string {
	return fmt.Sprintf(
		`{"tenantUUID":%q,"tenantToken":%q,"communicationEndpoints":[%q]}`,
		tenantUUID, tenantToken, endpoint,
	)
}
func connectionInfoEmptyEndpointsBody(tenantUUID, tenantToken string) string {
	return fmt.Sprintf(
		`{"tenantUUID":%q,"tenantToken":%q}`,
		tenantUUID, tenantToken,
	)
}
