// Copyright Dynatrace LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

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

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	dtversion "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tests for the version reconciler's auto-update flow.
//
// The version reconciler only modifies the DynaKube struct in memory and never
// writes to the cluster, so a fake client is enough — no envtest required.
//
// The HTTP-level response cache is exercised using real DT client
// implementations backed by an in-process fake transport (no network I/O).
//
// synctest.Test is used for cache-expiry subtests: both the cache-populating
// reconcile and the post-expiry reconcile run inside the same fake-clock
// bubble, so time.Sleep advances synthetic time instantly and the cache entry's
// creationTime is stamped with the same fake clock that isOutdated() checks.
// The fake transport avoids HTTP server goroutines that could interfere with
// synctest's goroutine tracking.

func TestAutoUpdateVersionReconciler(t *testing.T) {
	const (
		firstOneAgentVersion   = "1.2.3.4-11"
		updatedOneAgentVersion = "1.2.3.4-12"

		firstActiveGateVersion   = "1.2.3.4-56"
		updatedActiveGateVersion = "1.2.3.4-57"

		publicRegistryURIBase    = "registry.example.com/oneagent:"
		publicRegistryFirstTag   = "1.2.3.4-66"
		publicRegistryUpdatedTag = "1.2.3.4-67"
		publicRegistryFirstURI   = publicRegistryURIBase + publicRegistryFirstTag
		publicRegistryUpdatedURI = publicRegistryURIBase + publicRegistryUpdatedTag

		testCacheTTL = time.Minute
	)

	// OneAgent — tenant registry path (calls GetLatestAgentVersion).
	t.Run("OneAgent via tenant registry", func(t *testing.T) {
		t.Run("sets initial status from version client response", func(t *testing.T) {
			transport := newFakeTransport(agentVersionBody(firstOneAgentVersion))
			dk := newCloudNativeDynaKube()

			err := NewReconciler(fake.NewClient()).ReconcileOneAgent(t.Context(), dk, nil, newVersionClient(t, transport, testCacheTTL))
			require.NoError(t, err)

			assert.Equal(t, firstOneAgentVersion, dk.Status.OneAgent.Version)
		})

		t.Run("uses cached response within TTL", func(t *testing.T) {
			transport := newFakeTransport(agentVersionBody(firstOneAgentVersion))
			versionClient := newVersionClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newCloudNativeDynaKube()

			require.NoError(t, reconciler.ReconcileOneAgent(t.Context(), dk, nil, versionClient))
			require.NoError(t, reconciler.ReconcileOneAgent(t.Context(), dk, nil, versionClient))

			assert.Equal(t, firstOneAgentVersion, dk.Status.OneAgent.Version)
			transport.assertCalls(t, 1, "second reconcile must be served from cache")
		})

		t.Run("fetches fresh version after cache TTL expires", func(t *testing.T) {
			transport := newFakeTransport(
				agentVersionBody(firstOneAgentVersion),
				agentVersionBody(updatedOneAgentVersion),
			)
			versionClient := newVersionClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newCloudNativeDynaKube()

			synctest.Test(t, func(t *testing.T) {
				require.NoError(t, reconciler.ReconcileOneAgent(context.Background(), dk, nil, versionClient))
				require.Equal(t, firstOneAgentVersion, dk.Status.OneAgent.Version)

				time.Sleep(testCacheTTL + time.Second) // advance fake clock past the TTL

				require.NoError(t, reconciler.ReconcileOneAgent(context.Background(), dk, nil, versionClient))
			})

			assert.Equal(t, updatedOneAgentVersion, dk.Status.OneAgent.Version)
			transport.assertCalls(t, 2)
		})
	})

	// ActiveGate — tenant registry path (calls GetLatestActiveGateVersion).
	t.Run("ActiveGate via tenant registry", func(t *testing.T) {
		t.Run("sets initial status from version client response", func(t *testing.T) {
			transport := newFakeTransport(activeGateVersionBody(firstActiveGateVersion))
			dk := newActiveGateDynaKube()

			err := NewReconciler(fake.NewClient()).ReconcileActiveGate(t.Context(), dk, nil, newVersionClient(t, transport, testCacheTTL))
			require.NoError(t, err)

			assert.Equal(t, firstActiveGateVersion, dk.Status.ActiveGate.Version)
		})

		t.Run("uses cached response within TTL", func(t *testing.T) {
			transport := newFakeTransport(activeGateVersionBody(firstActiveGateVersion))
			versionClient := newVersionClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newActiveGateDynaKube()

			require.NoError(t, reconciler.ReconcileActiveGate(t.Context(), dk, nil, versionClient))
			require.NoError(t, reconciler.ReconcileActiveGate(t.Context(), dk, nil, versionClient))

			assert.Equal(t, firstActiveGateVersion, dk.Status.ActiveGate.Version)
			transport.assertCalls(t, 1, "second reconcile must be served from cache")
		})

		t.Run("fetches fresh version after cache TTL expires", func(t *testing.T) {
			transport := newFakeTransport(
				activeGateVersionBody(firstActiveGateVersion),
				activeGateVersionBody(updatedActiveGateVersion),
			)
			versionClient := newVersionClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newActiveGateDynaKube()

			synctest.Test(t, func(t *testing.T) {
				require.NoError(t, reconciler.ReconcileActiveGate(context.Background(), dk, nil, versionClient))
				require.Equal(t, firstActiveGateVersion, dk.Status.ActiveGate.Version)

				time.Sleep(testCacheTTL + time.Second)

				require.NoError(t, reconciler.ReconcileActiveGate(context.Background(), dk, nil, versionClient))
			})

			assert.Equal(t, updatedActiveGateVersion, dk.Status.ActiveGate.Version)
			transport.assertCalls(t, 2)
		})
	})

	// OneAgent — public registry path (calls GetComponentLatestInfo instead of
	// the version API; the image-discovery endpoint is also response-cached).
	t.Run("OneAgent via public registry", func(t *testing.T) {
		t.Run("sets initial status from image client response", func(t *testing.T) {
			transport := newFakeTransport(publicRegistryBody(dtimage.OneAgent, publicRegistryFirstURI))
			dk := newPublicRegistryDynaKube()

			err := NewReconciler(fake.NewClient()).ReconcileOneAgent(t.Context(), dk, newImageClient(t, transport, testCacheTTL), nil)
			require.NoError(t, err)

			assert.Equal(t, publicRegistryFirstTag, dk.Status.OneAgent.Version)
			assert.Equal(t, publicRegistryFirstURI, dk.Status.OneAgent.ImageID)
		})

		t.Run("uses cached response within TTL", func(t *testing.T) {
			transport := newFakeTransport(publicRegistryBody(dtimage.OneAgent, publicRegistryFirstURI))
			imageClient := newImageClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newPublicRegistryDynaKube()

			require.NoError(t, reconciler.ReconcileOneAgent(t.Context(), dk, imageClient, nil))
			require.NoError(t, reconciler.ReconcileOneAgent(t.Context(), dk, imageClient, nil))

			assert.Equal(t, publicRegistryFirstTag, dk.Status.OneAgent.Version)
			transport.assertCalls(t, 1, "second reconcile must be served from cache")
		})

		t.Run("fetches fresh image after cache TTL expires", func(t *testing.T) {
			transport := newFakeTransport(
				publicRegistryBody(dtimage.OneAgent, publicRegistryFirstURI),
				publicRegistryBody(dtimage.OneAgent, publicRegistryUpdatedURI),
			)
			imageClient := newImageClient(t, transport, testCacheTTL)
			reconciler := NewReconciler(fake.NewClient())
			dk := newPublicRegistryDynaKube()

			synctest.Test(t, func(t *testing.T) {
				require.NoError(t, reconciler.ReconcileOneAgent(context.Background(), dk, imageClient, nil))
				require.Equal(t, publicRegistryFirstTag, dk.Status.OneAgent.Version)

				time.Sleep(testCacheTTL + time.Second)

				require.NoError(t, reconciler.ReconcileOneAgent(context.Background(), dk, imageClient, nil))
			})

			assert.Equal(t, publicRegistryUpdatedTag, dk.Status.OneAgent.Version)
			assert.Equal(t, publicRegistryUpdatedURI, dk.Status.OneAgent.ImageID)
			transport.assertCalls(t, 2)
		})
	})
}

// --- DynaKube constructors -------------------------------------------------

// newCloudNativeDynaKube returns a CloudNativeFullStack DynaKube with no
// custom image or version, so IsAutoUpdateEnabled() returns true and the
// reconciler always calls the version API on every reconcile loop.
func newCloudNativeDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "test-dk", Namespace: testNamespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
	}
}

// newActiveGateDynaKube returns a DynaKube with one ActiveGate capability so
// that IsEnabled() returns true and the ActiveGate version is reconciled.
func newActiveGateDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "test-dk", Namespace: testNamespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName},
			},
		},
	}
}

// newPublicRegistryDynaKube returns a CloudNativeFullStack DynaKube with the
// platform token status set, causing the reconciler to call the
// image-discovery API instead of the version API.
func newPublicRegistryDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			APIToken: dynakube.APITokenStatus{
				Platform: new(true),
			},
		},
	}
}

// --- Client constructors ---------------------------------------------------

// newVersionClient builds a dtversion.Client whose HTTP transport is the given
// fakeTransport wrapped in a cache round-tripper with ttl.
// PaasToken is set to t.Name() so every subtest has a unique Authorization
// header, giving each subtest an isolated namespace in the global cache.
func newVersionClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) dtversion.Client {
	t.Helper()

	return dtversion.NewClient(core.NewClient(core.Config{
		BaseURL:    mustParseURL(t, "http://fake-dt.test"),
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		PaasToken:  t.Name(),
	}))
}

// newImageClient builds a dtimage.Client whose HTTP transport is the given
// fakeTransport wrapped in a cache round-tripper with ttl.
// APIToken is set to t.Name() for the same cache isolation reason.
func newImageClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) dtimage.Client {
	t.Helper()

	return dtimage.NewClient(core.NewClient(core.Config{
		BaseURL:    mustParseURL(t, "http://fake-dt.test"),
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		APIToken:   t.Name(),
	}))
}

// --- Fake transport --------------------------------------------------------

// fakeTransport is an in-process http.RoundTripper that serves preset JSON
// response bodies in sequence, repeating the last once the list is exhausted.
// It records the total call count for assertions.
// Being in-process avoids HTTP server goroutines that could interfere with
// synctest's fake clock and goroutine tracking.
type fakeTransport struct {
	calls  atomic.Int64
	bodies []string
}

func newFakeTransport(bodies ...string) *fakeTransport {
	return &fakeTransport{bodies: bodies}
}

func (ft *fakeTransport) assertCalls(t *testing.T, expected int64, msgAndArgs ...any) {
	t.Helper()
	assert.Equal(t, expected, ft.calls.Load(), msgAndArgs...)
}

func (ft *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	idx := int(ft.calls.Add(1)) - 1
	// this `min` makes sure we don't go out of bounds.
	// so if you have 2 bodies but 4 calls, the last 2 calls will get the 2. body as a response
	body := ft.bodies[min(idx, len(ft.bodies)-1)]

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

// --- Response body helpers -------------------------------------------------

func agentVersionBody(version string) string {
	return fmt.Sprintf(`{"latestAgentVersion":%q}`, version)
}

func activeGateVersionBody(version string) string {
	return fmt.Sprintf(`{"latestGatewayVersion":%q}`, version)
}

func publicRegistryBody(component dtimage.ComponentType, imageURI string) string {
	return fmt.Sprintf(`{"components":[{"type":%q,"imageUri":%q}]}`, component, imageURI)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	require.NoError(t, err)

	return u
}
