// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	dtversion "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tests for the kubemon statefulset reconciler's image-resolution auto-update flow.
//
// The reconciler writes a StatefulSet to the cluster but never reads back a "version" status,
// so the observable outcome is the container image on the StatefulSet spec. The fake client has
// no StatefulSet controller, so Reconcile always returns ErrRolloutInProgress after the first
// creation/update — this is expected and all assertions operate on the STS spec, not the error.
//
// The HTTP-level response cache is exercised using real DT client implementations backed by an
// in-process fake transport (no network I/O). synctest.Test is used for cache-expiry subtests
// so that time.Sleep advances synthetic time and the cache TTL check sees the same fake clock.

func TestReconcileAutoUpdate(t *testing.T) {
	const (
		autoUpdateAPIURL  = "https://fake-dt.test/api"
		autoUpdateAPIHost = "fake-dt.test"

		// 3-part versions so ToImageTag does not truncate, keeping first/updated images distinct.
		firstVersion   = "1.2.3"
		updatedVersion = "1.2.4"

		firstTenantImage   = autoUpdateAPIHost + kubemonapi.TenantRegistrySubPath + ":" + firstVersion + "-raw"
		updatedTenantImage = autoUpdateAPIHost + kubemonapi.TenantRegistrySubPath + ":" + updatedVersion + "-raw"

		publicFirstURI   = "public.registry.example.com/activegate:" + firstVersion
		publicUpdatedURI = "public.registry.example.com/activegate:" + updatedVersion

		testAutoUpdateCacheTTL = time.Minute
	)

	// tenant registry — version client is called; GetDefaultImage builds the image from APIURL host.
	t.Run("tenant registry", func(t *testing.T) {
		t.Run("uses version client to build container image", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(activeGateVersionBody(firstVersion))
			dk := newAutoUpdateKubemonDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, nil, newVersionClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL)),
				k8sstatefulset.ErrRolloutInProgress,
			)

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, firstTenantImage, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 1)
		})

		t.Run("uses cached response within TTL", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(activeGateVersionBody(firstVersion))
			versionClient := newVersionClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL)
			dk := newAutoUpdateKubemonDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			require.ErrorIs(t, reconciler.Reconcile(t.Context(), dk, nil, versionClient), k8sstatefulset.ErrRolloutInProgress)
			require.ErrorIs(t, reconciler.Reconcile(t.Context(), dk, nil, versionClient), k8sstatefulset.ErrRolloutInProgress)

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, firstTenantImage, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 1, "second reconcile must be served from cache")
		})

		t.Run("fetches fresh version after cache TTL expires", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(
				activeGateVersionBody(firstVersion),
				activeGateVersionBody(updatedVersion),
			)
			versionClient := newVersionClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL)
			dk := newAutoUpdateKubemonDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			synctest.Test(t, func(t *testing.T) {
				require.ErrorIs(t, reconciler.Reconcile(context.Background(), dk, nil, versionClient), k8sstatefulset.ErrRolloutInProgress)

				time.Sleep(testAutoUpdateCacheTTL + time.Second) // advance fake clock past TTL

				require.ErrorIs(t, reconciler.Reconcile(context.Background(), dk, nil, versionClient), k8sstatefulset.ErrRolloutInProgress)
			})

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, updatedTenantImage, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 2)
		})
	})

	// public registry — image client is called; the returned URI is set directly as the container image.
	t.Run("public registry", func(t *testing.T) {
		t.Run("uses image client to build container image", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(activeGatePublicRegistryBody(publicFirstURI))
			dk := newAutoUpdatePublicRegistryDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, newImageClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL), nil),
				k8sstatefulset.ErrRolloutInProgress,
			)

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, publicFirstURI, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 1)
		})

		t.Run("uses cached response within TTL", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(activeGatePublicRegistryBody(publicFirstURI))
			imageClient := newImageClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL)
			dk := newAutoUpdatePublicRegistryDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			require.ErrorIs(t, reconciler.Reconcile(t.Context(), dk, imageClient, nil), k8sstatefulset.ErrRolloutInProgress)
			require.ErrorIs(t, reconciler.Reconcile(t.Context(), dk, imageClient, nil), k8sstatefulset.ErrRolloutInProgress)

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, publicFirstURI, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 1, "second reconcile must be served from cache")
		})

		t.Run("fetches fresh image after cache TTL expires", func(t *testing.T) {
			transport := newFakeAutoUpdateTransport(
				activeGatePublicRegistryBody(publicFirstURI),
				activeGatePublicRegistryBody(publicUpdatedURI),
			)
			imageClient := newImageClientForAutoUpdate(t, transport, testAutoUpdateCacheTTL)
			dk := newAutoUpdatePublicRegistryDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			synctest.Test(t, func(t *testing.T) {
				require.ErrorIs(t, reconciler.Reconcile(context.Background(), dk, imageClient, nil), k8sstatefulset.ErrRolloutInProgress)

				time.Sleep(testAutoUpdateCacheTTL + time.Second)

				require.ErrorIs(t, reconciler.Reconcile(context.Background(), dk, imageClient, nil), k8sstatefulset.ErrRolloutInProgress)
			})

			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, publicUpdatedURI, sts.Spec.Template.Spec.Containers[0].Image)
			transport.assertCalls(t, 2)
		})
	})

	// switching registry source — the StatefulSet image must update when the registry source changes.
	t.Run("switching registry source", func(t *testing.T) {
		t.Run("from tenant to public registry updates container image", func(t *testing.T) {
			tenantTransport := newFakeAutoUpdateTransport(activeGateVersionBody(firstVersion))
			publicTransport := newFakeAutoUpdateTransport(activeGatePublicRegistryBody(publicFirstURI))

			dk := newAutoUpdateKubemonDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			// Phase 1: tenant registry — version client resolves the image
			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, nil, newVersionClientForAutoUpdate(t, tenantTransport, testAutoUpdateCacheTTL)),
				k8sstatefulset.ErrRolloutInProgress,
			)
			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, firstTenantImage, sts.Spec.Template.Spec.Containers[0].Image)

			// Phase 2: add public-registry annotation — image client resolves the image instead
			dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, newImageClientForAutoUpdate(t, publicTransport, testAutoUpdateCacheTTL), nil),
				k8sstatefulset.ErrRolloutInProgress,
			)
			sts = requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, publicFirstURI, sts.Spec.Template.Spec.Containers[0].Image)
		})

		t.Run("from public to tenant registry updates container image", func(t *testing.T) {
			publicTransport := newFakeAutoUpdateTransport(activeGatePublicRegistryBody(publicFirstURI))
			tenantTransport := newFakeAutoUpdateTransport(activeGateVersionBody(firstVersion))

			dk := newAutoUpdatePublicRegistryDynaKube(autoUpdateAPIURL)
			fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
			reconciler := statefulset.NewReconciler(fakeClient)

			// Phase 1: public registry — image client resolves the image
			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, newImageClientForAutoUpdate(t, publicTransport, testAutoUpdateCacheTTL), nil),
				k8sstatefulset.ErrRolloutInProgress,
			)
			sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, publicFirstURI, sts.Spec.Template.Spec.Containers[0].Image)

			// Phase 2: remove public-registry annotation — version client resolves the image instead
			dk.Annotations = nil
			require.ErrorIs(t,
				reconciler.Reconcile(t.Context(), dk, nil, newVersionClientForAutoUpdate(t, tenantTransport, testAutoUpdateCacheTTL)),
				k8sstatefulset.ErrRolloutInProgress,
			)
			sts = requireTestStatefulSet(t, t.Context(), fakeClient, dk)
			assert.Equal(t, firstTenantImage, sts.Spec.Template.Spec.Containers[0].Image)
		})
	})
}

// --- DynaKube constructors ---------------------------------------------------

// newAutoUpdateKubemonDynaKube returns a DynaKube with KubernetesMonitoring enabled and no custom
// image, so the reconciler always calls the version client to resolve the container image.
func newAutoUpdateKubemonDynaKube(apiURL string) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "autoupdate-dk",
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:               apiURL,
			KubernetesMonitoring: &kubemonapi.Spec{},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: integrationKubeSystemUUID,
		},
	}
}

// newAutoUpdatePublicRegistryDynaKube returns a DynaKube with the public-registry feature flag
// enabled, causing the reconciler to call the image client instead of the version client.
func newAutoUpdatePublicRegistryDynaKube(apiURL string) *dynakube.DynaKube {
	dk := newAutoUpdateKubemonDynaKube(apiURL)
	dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}

	return dk
}

// --- Client constructors -----------------------------------------------------

// newVersionClientForAutoUpdate builds a real dtversion.Client whose HTTP transport is the given
// fakeAutoUpdateTransport wrapped in a cache round-tripper with the specified TTL.
// PaasToken is set to t.Name() so each subtest has an isolated namespace in the global cache.
func newVersionClientForAutoUpdate(t *testing.T, transport http.RoundTripper, ttl time.Duration) dtversion.Client {
	t.Helper()

	return dtversion.NewClient(core.NewClient(core.Config{
		BaseURL:    mustParseURL(t, "http://fake-dt.test"),
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		PaasToken:  t.Name(),
	}))
}

// newImageClientForAutoUpdate builds a real dtimage.Client whose HTTP transport is the given
// fakeAutoUpdateTransport wrapped in a cache round-tripper with the specified TTL.
// APIToken is set to t.Name() for the same cache-isolation reason.
func newImageClientForAutoUpdate(t *testing.T, transport http.RoundTripper, ttl time.Duration) dtimage.Client {
	t.Helper()

	return dtimage.NewClient(core.NewClient(core.Config{
		BaseURL:    mustParseURL(t, "http://fake-dt.test"),
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		APIToken:   t.Name(),
	}))
}

// --- Fake transport ----------------------------------------------------------

// fakeTransport is an in-process http.RoundTripper that serves preset JSON response
// bodies in sequence, repeating the last once the list is exhausted. It records the total call
// count for assertions. Being in-process avoids HTTP server goroutines that could interfere with
// synctest's fake clock and goroutine tracking.
type fakeTransport struct {
	calls  atomic.Int64
	bodies []string
}

func newFakeAutoUpdateTransport(bodies ...string) *fakeTransport {
	return &fakeTransport{bodies: bodies}
}

func (ft *fakeTransport) assertCalls(t *testing.T, expected int64, msgAndArgs ...any) {
	t.Helper()
	assert.Equal(t, expected, ft.calls.Load(), msgAndArgs...)
}

func (ft *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	idx := int(ft.calls.Add(1)) - 1
	body := ft.bodies[min(idx, len(ft.bodies)-1)]

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

// --- Response body helpers ---------------------------------------------------

func activeGateVersionBody(version string) string {
	return fmt.Sprintf(`{"latestGatewayVersion":%q}`, version)
}

func activeGatePublicRegistryBody(imageURI string) string {
	return fmt.Sprintf(`{"components":[{"type":%q,"imageUri":%q}]}`, dtimage.ActiveGate, imageURI)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	require.NoError(t, err)

	return u
}
