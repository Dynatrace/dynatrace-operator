// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/oci/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the EdgeConnect version reconciler's auto update flow.
//
// The fleet management side uses the real dtimage.Client backed by a fake transport, so
// the assertions cover the whole path down to the HTTP request, including the registry query
// parameter. The OCI registry fallback is mocked, because it talks to a container registry through
// go containerregistry rather than through the Dynatrace API client.
//
// Time is driven by a frozen timeprovider.Provider, because
// minRequestThreshold is the throttle under test: reconciling twice without advancing the provider
// has to hit the throttle, advancing past it has to release it.

const (
	firstImageURI   = "docker.io/dynatrace/edgeconnect:1.2.3@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	updatedImageURI = "docker.io/dynatrace/edgeconnect:1.2.4@sha256:2d8ef3799ee82ea6b897ed1b96102f3838c2de4f96a812fe1e708a731c9f9966"

	overrideRegistry = "my.registry.io"
	overrideImageURI = "my.registry.io/dynatrace/edgeconnect:1.2.3@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

	otherRegistry = "other.registry.io"
	otherImageURI = "other.registry.io/dynatrace/edgeconnect:1.2.3@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
)

func TestAutoUpdateVersionReconciler(t *testing.T) {
	t.Run("auto update enabled", func(t *testing.T) {
		t.Run("sets initial status from the fleet management response", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI))
			ec := newAutoUpdateEdgeConnect(t, true)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, firstImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, "1.2.3", ec.Status.Version.Version)
			assert.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
			transport.assertCalls(t, 1)
		})

		t.Run("does not probe again within the threshold", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI), containerImagesBody(updatedImageURI))
			ec := newAutoUpdateEdgeConnect(t, true)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, firstImageURI, ec.Status.Version.ImageID)
			transport.assertCalls(t, 1, "second reconcile must be throttled by minRequestThreshold")
		})

		t.Run("probes again once the threshold expired", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI), containerImagesBody(updatedImageURI))
			ec := newAutoUpdateEdgeConnect(t, true)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.Equal(t, firstImageURI, ec.Status.Version.ImageID)

			now.Set(now.Now().Add(minRequestThreshold + time.Second))
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, updatedImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, "1.2.4", ec.Status.Version.Version)
			transport.assertCalls(t, 2)
		})
	})

	t.Run("auto update disabled", func(t *testing.T) {
		t.Run("still resolves the initial image", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, firstImageURI, ec.Status.Version.ImageID)
			transport.assertCalls(t, 1, "an EdgeConnect without an image has to resolve one regardless of auto update")
		})

		t.Run("does not probe again once the threshold expired", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI), containerImagesBody(updatedImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))

			now.Set(now.Now().Add(minRequestThreshold + time.Second))
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, firstImageURI, ec.Status.Version.ImageID)
			transport.assertCalls(t, 1, "auto update is disabled, so an expired threshold must not trigger a probe")
		})
	})

	// A publicRegistryOverride has to be applied right away, otherwise a user without auto update
	// could never move an EdgeConnect to or away from their own registry.
	t.Run("publicRegistryOverride without auto update", func(t *testing.T) {
		t.Run("is passed to fleet management as a query parameter", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(overrideImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			ec.Spec.PublicRegistryOverride = overrideRegistry
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, overrideImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, status.PublicRegistryWithOverrideVersionSource, ec.Status.Version.Source)
			assert.Equal(t, overrideRegistry, transport.queryParam(t, 0, "registry"))
		})

		t.Run("adding it probes again", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(firstImageURI), containerImagesBody(overrideImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.Equal(t, firstImageURI, ec.Status.Version.ImageID)

			ec.Spec.PublicRegistryOverride = overrideRegistry
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, overrideImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, overrideRegistry, transport.queryParam(t, 1, "registry"))
			transport.assertCalls(t, 2, "adding an override must not wait for the threshold")
		})

		t.Run("removing it probes again", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(overrideImageURI), containerImagesBody(firstImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			ec.Spec.PublicRegistryOverride = overrideRegistry
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.Equal(t, overrideImageURI, ec.Status.Version.ImageID)

			ec.Spec.PublicRegistryOverride = ""
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, firstImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
			assert.Empty(t, transport.queryParam(t, 1, "registry"), "the override is gone, so the registry must not be requested")
			transport.assertCalls(t, 2, "removing an override must not wait for the threshold")
		})

		t.Run("switching it to another registry probes again", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(overrideImageURI), containerImagesBody(otherImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			ec.Spec.PublicRegistryOverride = overrideRegistry
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.Equal(t, overrideImageURI, ec.Status.Version.ImageID)

			ec.Spec.PublicRegistryOverride = otherRegistry
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, otherImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, otherRegistry, transport.queryParam(t, 1, "registry"))
			transport.assertCalls(t, 2, "switching an override must not wait for the threshold")
		})

		t.Run("leaving it unchanged does not probe again", func(t *testing.T) {
			transport := newFakeTransport(containerImagesBody(overrideImageURI))
			ec := newAutoUpdateEdgeConnect(t, false)
			ec.Spec.PublicRegistryOverride = overrideRegistry
			now := timeprovider.New().Freeze()

			require.NoError(t, reconcile(t, ec, transport, now, nil))
			require.NoError(t, reconcile(t, ec, transport, now, nil))

			assert.Equal(t, overrideImageURI, ec.Status.Version.ImageID)
			transport.assertCalls(t, 1, "an unchanged override must not force a probe")
		})
	})

	t.Run("fleet management unavailable", func(t *testing.T) {
		t.Run("falls back to the OCI registry", func(t *testing.T) {
			transport := newFakeTransport(serverErrorBody())
			ec := newAutoUpdateEdgeConnect(t, true)
			now := timeprovider.New().Freeze()

			fakeRegistry := registrymock.NewImageGetter(t)
			fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).Return(fakeRegistryImageVersion(), nil)

			require.NoError(t, reconcile(t, ec, transport, now, fakeRegistry))

			assert.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
			assert.Equal(t, fakeImageVersion, ec.Status.Version.Version)
			assert.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
			transport.assertCalls(t, 1)
		})

		t.Run("fails the reconcile when the OCI registry is unavailable too", func(t *testing.T) {
			transport := newFakeTransport(serverErrorBody())
			ec := newAutoUpdateEdgeConnect(t, true)
			now := timeprovider.New().Freeze()

			fakeRegistry := registrymock.NewImageGetter(t)
			fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
				Return(registry.ImageVersion{}, assert.AnError)

			require.Error(t, reconcile(t, ec, transport, now, fakeRegistry))
			assert.Empty(t, ec.Status.Version.ImageID)
		})
	})

	t.Run("custom image never contacts fleet management", func(t *testing.T) {
		transport := newFakeTransport(containerImagesBody(firstImageURI))
		ec := newAutoUpdateEdgeConnect(t, true)
		ec.Spec.ImageRef.Repository = "my.registry.io/custom/edgeconnect"
		ec.Spec.ImageRef.Tag = "4.5.6"
		now := timeprovider.New().Freeze()

		require.NoError(t, reconcile(t, ec, transport, now, nil))

		assert.Equal(t, "my.registry.io/custom/edgeconnect:4.5.6", ec.Status.Version.ImageID)
		assert.Equal(t, status.CustomImageVersionSource, ec.Status.Version.Source)
		transport.assertCalls(t, 0, "a custom image is taken over as-is")
	})
}

// --- test harness ----------------------------------------------------------

// reconcile runs one full version reconcile the way the controller does, building a new Reconciler
// each time. registryClient may be nil when the OCI registry fallback is not expected to be used.
func reconcile(t *testing.T, ec *edgeconnect.EdgeConnect, transport http.RoundTripper, now *timeprovider.Provider, registryClient *registrymock.ImageGetter) error {
	t.Helper()

	registryClientProvider := failingRegistryClientProvider(t)
	if registryClient != nil {
		registryClientProvider = staticRegistryClientProvider(registryClient)
	}

	reconciler := NewReconciler(
		fake.NewClient(),
		staticImageClientProvider(newImageClient(t, transport)),
		registryClientProvider,
		now,
		ec,
	)

	return reconciler.Reconcile(t.Context())
}

func newAutoUpdateEdgeConnect(t *testing.T, autoUpdate bool) *edgeconnect.EdgeConnect {
	t.Helper()

	ec := createBasicEdgeConnect(t)
	ec.Spec.AutoUpdate = &autoUpdate

	return ec
}

// newImageClient builds the real fleet management image client on top of an in-process transport, so
// the request the reconciler produces is asserted rather than mocked away.
func newImageClient(t *testing.T, transport http.RoundTripper) dtimage.Client {
	t.Helper()

	return dtimage.NewClient(core.NewClient(core.Config{
		BaseURL:    mustParseURL(t, "http://fake-dt.test"),
		HTTPClient: &http.Client{Transport: transport},
	}))
}

// --- fake transport --------------------------------------------------------

// fakeTransport is an in-process http.RoundTripper that serves preset responses in sequence,
// repeating the last one once the list is exhausted. It records every request URL so tests can
// assert on the query parameters the reconciler sent.
type fakeTransport struct {
	mutex     sync.Mutex
	responses []fakeResponse
	requests  []*url.URL
}

type fakeResponse struct {
	body   string
	status int
}

func newFakeTransport(responses ...fakeResponse) *fakeTransport {
	return &fakeTransport{responses: responses}
}

func (ft *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	idx := len(ft.requests)
	ft.requests = append(ft.requests, r.URL)

	// stay on the last response once the list is exhausted
	response := ft.responses[min(idx, len(ft.responses)-1)]

	return &http.Response{
		StatusCode: response.status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(response.body)),
		Request:    r,
	}, nil
}

func (ft *fakeTransport) assertCalls(t *testing.T, expected int, msgAndArgs ...any) {
	t.Helper()
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	assert.Len(t, ft.requests, expected, msgAndArgs...)
}

// queryParam returns the value the given request sent for name, so tests can assert that
// publicRegistryOverride reached the endpoint.
func (ft *fakeTransport) queryParam(t *testing.T, requestIdx int, name string) string {
	t.Helper()
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	require.Greater(t, len(ft.requests), requestIdx, "no request recorded at index %d", requestIdx)

	return ft.requests[requestIdx].Query().Get(name)
}

// --- response body helpers -------------------------------------------------

func containerImagesBody(imageURI string) fakeResponse {
	return fakeResponse{
		status: http.StatusOK,
		body:   fmt.Sprintf(`{"components":[{"type":%q,"imageUri":%q}]}`, dtimage.EdgeConnect, imageURI),
	}
}

func serverErrorBody() fakeResponse {
	return fakeResponse{
		status: http.StatusInternalServerError,
		body:   `{"error":{"code":500,"message":"fleet management unavailable"}}`,
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	return parsed
}
