// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package activegate

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
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConnectionInfoCacheTTL = time.Minute

func TestReconcile_ConnectionInfoCache(t *testing.T) {
	t.Run("uses cached connection info within TTL", func(t *testing.T) {
		transport := newFakeCITransport(
			agConnectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
		)
		agClient := newConnectionInfoAGClient(t, transport, testConnectionInfoCacheTTL)
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, fakeClient)

		require.NoError(t, r.Reconcile(t.Context(), agClient, dk))
		require.NoError(t, r.Reconcile(t.Context(), agClient, dk))

		transport.assertCalls(t, 1, "second reconcile must be served from cache")
	})

	t.Run("fetches fresh connection info after cache TTL expires", func(t *testing.T) {
		const updatedUUID = "updated-uuid"
		transport := newFakeCITransport(
			agConnectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
			agConnectionInfoBody(updatedUUID, testTenantToken, testTenantEndpoints),
		)
		agClient := newConnectionInfoAGClient(t, transport, testConnectionInfoCacheTTL)
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, fakeClient)

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, r.Reconcile(t.Context(), agClient, dk))

			time.Sleep(testConnectionInfoCacheTTL + time.Second)

			require.NoError(t, r.Reconcile(t.Context(), agClient, dk))
		})

		assert.Equal(t, updatedUUID, dk.Status.ActiveGate.ConnectionInfo.TenantUUID)
		transport.assertCalls(t, 2)
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

func newConnectionInfoAGClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) agclient.Client {
	t.Helper()

	u, err := url.Parse("https://fake-dt.test")
	require.NoError(t, err)

	return agclient.NewClient(core.NewClient(core.Config{
		BaseURL:    u,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		PaasToken:  t.Name(),
	}))
}

func agConnectionInfoBody(tenantUUID, tenantToken, endpoints string) string {
	return fmt.Sprintf(
		`{"tenantUUID":%q,"tenantToken":%q,"communicationEndpoints":%q}`,
		tenantUUID, tenantToken, endpoints,
	)
}
