// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/oci/registry"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fakeImageURI     = "docker.io/dynatrace/edgeconnect:latest@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageDigest  = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion = "1.2.3"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func staticImageClientProvider(imagesClient dtimage.Client) ImageClientProvider {
	return func(context.Context) (dtimage.Client, error) {
		return imagesClient, nil
	}
}

// failingImageClientProvider fails the test if an image client is requested at all, which asserts
// that no OAuth token exchange happens when no image has to be resolved.
func failingImageClientProvider(t *testing.T) ImageClientProvider {
	t.Helper()

	return func(context.Context) (dtimage.Client, error) {
		require.FailNow(t, "image client must not be built")

		return nil, nil
	}
}

func staticRegistryClientProvider(registryClient registry.ImageGetter) RegistryClientProvider {
	return func(context.Context) (registry.ImageGetter, error) {
		return registryClient, nil
	}
}

// failingRegistryClientProvider fails the test if a registry client is requested at all, which
// asserts that no pull secret is read when the OCI registry fallback is not taken.
func failingRegistryClientProvider(t *testing.T) RegistryClientProvider {
	t.Helper()

	return func(context.Context) (registry.ImageGetter, error) {
		require.FailNow(t, "registry client must not be built")

		return nil, nil
	}
}

func fakeRegistryImageVersion() registry.ImageVersion {
	return registry.ImageVersion{Digest: digest.Digest(fakeImageDigest), Version: fakeImageVersion}
}

func Test_updater_Update(t *testing.T) {
	t.Run("default image => fleet management used without override", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "").
			Return(&dtimage.Info{URI: fakeImageURI, Tag: fakeImageVersion}, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), failingRegistryClientProvider(t), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
		require.Equal(t, fakeImageVersion, ec.Status.Version.Version)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("publicRegistryOverride set => fleet management used with override", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)
		ec.Spec.PublicRegistryOverride = "my.registry.io"
		overrideURI := "my.registry.io/dynatrace/edgeconnect:latest@sha256:abc"

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "my.registry.io").
			Return(&dtimage.Info{URI: overrideURI, Tag: fakeImageVersion}, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), failingRegistryClientProvider(t), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, overrideURI, ec.Status.Version.ImageID)
		require.Equal(t, fakeImageVersion, ec.Status.Version.Version)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("fleet management fails => falls back to OCI registry", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "").
			Return(nil, errors.New("fleet management unavailable"))

		fakeRegistry := registrymock.NewImageGetter(t)
		fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
			Return(fakeRegistryImageVersion(), nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), staticRegistryClientProvider(fakeRegistry), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
		require.Equal(t, fakeImageVersion, ec.Status.Version.Version)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("image client cannot be built => falls back to OCI registry", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)

		erroringProvider := func(context.Context) (dtimage.Client, error) {
			return nil, errors.New("oauth token exchange failed")
		}

		fakeRegistry := registrymock.NewImageGetter(t)
		fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
			Return(fakeRegistryImageVersion(), nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), erroringProvider, staticRegistryClientProvider(fakeRegistry), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
		require.Equal(t, fakeImageVersion, ec.Status.Version.Version)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("fleet management returns no image => falls back to OCI registry", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "").
			Return(nil, nil)

		fakeRegistry := registrymock.NewImageGetter(t)
		fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
			Return(fakeRegistryImageVersion(), nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), staticRegistryClientProvider(fakeRegistry), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
		require.Equal(t, fakeImageVersion, ec.Status.Version.Version)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("publicRegistryOverride set and fleet management fails => falls back to the default registry", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)
		ec.Spec.PublicRegistryOverride = "my.registry.io"

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "my.registry.io").
			Return(nil, errors.New("fleet management unavailable"))

		// the fallback can only resolve the default image, so the override is not taken into accoutn
		fakeRegistry := registrymock.NewImageGetter(t)
		fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
			Return(fakeRegistryImageVersion(), nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), staticRegistryClientProvider(fakeRegistry), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
		require.NotContains(t, ec.Status.Version.ImageID, "my.registry.io")
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("both fleet management and OCI registry fail => error", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)

		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "").
			Return(nil, errors.New("fleet management unavailable"))

		fakeRegistry := registrymock.NewImageGetter(t)
		fakeRegistry.EXPECT().GetImageVersion(anyCtx, ec.Image()).
			Return(registry.ImageVersion{}, errors.New("registry unreachable"))

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), staticRegistryClientProvider(fakeRegistry), ec)

		require.Error(t, updater.Update(ctx))
	})

	t.Run("custom imageRef set => neither fleet management nor OCI registry used", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)
		ec.Spec.ImageRef.Repository = "my.registry.io/custom/edgeconnect"

		updater := newUpdater(fake.NewClient(), timeprovider.New(), failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, "my.registry.io/custom/edgeconnect:latest", ec.Status.Version.ImageID)
		require.Equal(t, string(status.CustomImageVersionSource), ec.Status.Version.Version)
		require.Equal(t, status.CustomImageVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})
}

func Test_updater_RequiresReconcile(t *testing.T) {
	currentTime := timeprovider.New().Freeze()

	t.Run("initial reconcile always required", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(true)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.NewTime(currentTime.Now().Add(-time.Hour)))
		ec.Spec.AutoUpdate = new(true)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("no reconcile if auto update is disabled and time is up", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.NewTime(currentTime.Now().Add(-time.Hour)))
		ec.Spec.AutoUpdate = new(false)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("no reconcile for a custom image, even with auto update and time up", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Spec.ImageRef = image.Ref{Repository: "my.registry.io/custom/edgeconnect", Tag: "1.2.3"}
		ec.Spec.AutoUpdate = new(true)
		ec.Status.Version.LastProbeTimestamp = new(metav1.NewTime(currentTime.Now().Add(-time.Hour)))
		ec.Status.Version.ImageID = ec.Image()
		ec.Status.Version.Source = status.CustomImageVersionSource

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Status.Version.ImageID = ec.Image()
		ec.Status.Version.Source = status.CustomImageVersionSource
		ec.Spec.ImageRef = image.Ref{
			Repository: "docker.io/dynatrace/superfancynew",
		}

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if switched away from a custom image", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(false)
		ec.Status.Version.ImageID = "docker.io/dynatrace/custom:1.2.3"
		ec.Status.Version.Source = status.CustomImageVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if publicRegistryOverride was added, even without auto update", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(false)
		ec.Spec.PublicRegistryOverride = "my.registry.io"
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("no reconcile if the image already comes from publicRegistryOverride", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), failingRegistryClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(false)
		ec.Spec.PublicRegistryOverride = "my.registry.io"
		ec.Status.Version.ImageID = "my.registry.io/dynatrace/edgeconnect:1.2.3"
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.False(t, updater.RequiresReconcile())
	})
}

func createBasicEdgeConnect(t *testing.T) *edgeconnect.EdgeConnect {
	t.Helper()

	return &edgeconnect.EdgeConnect{
		Spec: edgeconnect.EdgeConnectSpec{
			APIServer: "superfancy.dev.apps.dynatracelabs.com",
		},
		Status: edgeconnect.EdgeConnectStatus{},
	}
}
