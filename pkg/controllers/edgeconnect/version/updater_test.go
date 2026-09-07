// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fakeImageURI = "docker.io/dynatrace/edgeconnect:latest@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
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

func Test_updater_Update(t *testing.T) {
	t.Run("default image => fleet management used without override", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)
		fakeImageClient := imagemock.NewClient(t)
		fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, "").
			Return(&dtimage.Info{URI: fakeImageURI}, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, fakeImageURI, ec.Status.Version.ImageID)
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
			Return(&dtimage.Info{URI: overrideURI}, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), staticImageClientProvider(fakeImageClient), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, overrideURI, ec.Status.Version.ImageID)
		require.Equal(t, status.PublicRegistryVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("custom imageRef set => fleet management NOT used", func(t *testing.T) {
		ctx := t.Context()
		ec := createBasicEdgeConnect(t)
		ec.Spec.ImageRef.Repository = "my.registry.io/custom/edgeconnect"

		updater := newUpdater(fake.NewClient(), timeprovider.New(), failingImageClientProvider(t), ec)

		require.NoError(t, updater.Update(ctx))
		require.Equal(t, "my.registry.io/custom/edgeconnect:latest", ec.Status.Version.ImageID)
		require.Equal(t, status.CustomImageVersionSource, ec.Status.Version.Source)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})
}

func Test_updater_RequiresReconcile(t *testing.T) {
	currentTime := timeprovider.New().Freeze()

	t.Run("initial reconcile always required", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(true)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.NewTime(currentTime.Now().Add(-time.Hour)))
		ec.Spec.AutoUpdate = new(true)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("no reconcile if auto update is disabled and time is up", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.NewTime(currentTime.Now().Add(-time.Hour)))
		ec.Spec.AutoUpdate = new(false)
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

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
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(false)
		ec.Status.Version.ImageID = "docker.io/dynatrace/custom:1.2.3"
		ec.Status.Version.Source = status.CustomImageVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if publicRegistryOverride was added, even without auto update", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

		ec.Status.Version.LastProbeTimestamp = new(metav1.Now())
		ec.Spec.AutoUpdate = new(false)
		ec.Spec.PublicRegistryOverride = "my.registry.io"
		ec.Status.Version.ImageID = fakeImageURI
		ec.Status.Version.Source = status.PublicRegistryVersionSource

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("no reconcile if the image already comes from publicRegistryOverride", func(t *testing.T) {
		ec := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, failingImageClientProvider(t), ec)

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
