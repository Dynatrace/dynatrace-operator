// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package activegate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		recorder := newResponseRecorder(
			agConnectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
		)
		s := httptest.NewTestServer(t, recorder)
		agClient := newConnectionInfoAGClient(t, s.Client().Transport, testConnectionInfoCacheTTL)
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, fakeClient)

		require.NoError(t, r.Reconcile(t.Context(), agClient, dk))
		require.NoError(t, r.Reconcile(t.Context(), agClient, dk))

		assert.Equal(t, 1, recorder.calls, "second reconcile must be served from cache")
	})

	t.Run("fetches fresh connection info after cache TTL expires", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const updatedUUID = "updated-uuid"
			recorder := newResponseRecorder(
				agConnectionInfoBody(testTenantUUID, testTenantToken, testTenantEndpoints),
				agConnectionInfoBody(updatedUUID, testTenantToken, testTenantEndpoints),
			)
			s := httptest.NewTestServer(t, recorder)
			agClient := newConnectionInfoAGClient(t, s.Client().Transport, testConnectionInfoCacheTTL)
			dk := getTestDynakube()
			fakeClient := fake.NewClient(dk)
			r := NewReconciler(fakeClient, fakeClient)

			require.NoError(t, r.Reconcile(t.Context(), agClient, dk))

			time.Sleep(testConnectionInfoCacheTTL + time.Second)

			require.NoError(t, r.Reconcile(t.Context(), agClient, dk))

			assert.Equal(t, updatedUUID, dk.Status.ActiveGate.ConnectionInfo.TenantUUID)
			assert.Equal(t, 2, recorder.calls)
		})
	})
}

type responseRecorder struct {
	calls  int
	bodies []string
}

func newResponseRecorder(bodies ...string) *responseRecorder {
	return &responseRecorder{bodies: bodies}
}

func (r *responseRecorder) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	idx := r.calls
	body := r.bodies[min(idx, len(r.bodies)-1)]
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(body))
	r.calls++
}

func newConnectionInfoAGClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) agclient.Client {
	t.Helper()

	u, err := url.Parse("https://fake-dt.test")
	require.NoError(t, err)

	return agclient.NewClient(core.NewClient(core.Config{
		BaseURL:    u,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		// Set to t.Name() so each subtest has an isolated namespace in the global cache.
		PaasToken: t.Name(),
	}))
}

func agConnectionInfoBody(tenantUUID, tenantToken, endpoints string) string {
	return fmt.Sprintf(
		`{"tenantUUID":%q,"tenantToken":%q,"communicationEndpoints":%q}`,
		tenantUUID, tenantToken, endpoints,
	)
}
