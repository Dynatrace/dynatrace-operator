// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/oci/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fakeDigest           = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	expectedDefaultImage = "docker.io/dynatrace/edgeconnect:latest@" + fakeDigest
)

func Test_updater_Update(t *testing.T) {
	t.Run("default image => registry used", func(t *testing.T) {
		ctx := t.Context()
		edgeConnect := createBasicEdgeConnect(t)
		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

		err := updater.Update(ctx)
		require.NoError(t, err)

		require.Equal(t, expectedDefaultImage, edgeConnect.Status.Version.ImageID)
		require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)

		// check invalid digest
		invalidImageVersion := registry.ImageVersion{Digest: "invaliddigest"}
		fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(invalidImageVersion, nil)

		updater = newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

		err = updater.Update(ctx)
		require.NoError(t, err)

		// digest should not have been updated due to probe timestamp
		require.Contains(t, edgeConnect.Status.Version.ImageID, fakeDigest)
	})

	t.Run("custom tag used => registry still used", func(t *testing.T) {
		ctx := t.Context()
		edgeConnect := createBasicEdgeConnect(t)
		customTag := "1.2.3"
		edgeConnect.Spec.ImageRef.Tag = customTag
		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

		err := updater.Update(ctx)
		require.NoError(t, err)

		require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:%s@%s", customTag, fakeDigest), edgeConnect.Status.Version.ImageID)
		require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)
	})

	t.Run("custom registry used => registry NOT used", func(t *testing.T) {
		ctx := t.Context()
		edgeConnect := createBasicEdgeConnect(t)
		customRegistry := "best.registry.io/dynatrace/edgeconnect"
		edgeConnect.Spec.ImageRef.Repository = customRegistry

		updater := newUpdater(fake.NewClient(), timeprovider.New(), nil, edgeConnect)

		err := updater.Update(ctx)
		require.NoError(t, err)

		require.Equal(t, customRegistry+":latest", edgeConnect.Status.Version.ImageID)
		require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)
	})
}

func Test_updater_combineImageWithDigest(t *testing.T) {
	edgeConnect := createBasicEdgeConnect(t)
	fakeRegistryClient := registrymock.NewImageGetter(t)

	updater := newUpdater(fake.NewClient(), nil, fakeRegistryClient, edgeConnect)

	t.Run("image and digest should be combined", func(t *testing.T) {
		combined, err := updater.combineImageWithDigest(t.Context(), fakeDigest)

		require.NoError(t, err)
		require.Equal(t, expectedDefaultImage, combined)
	})

	t.Run("malformed image should fail", func(t *testing.T) {
		edgeConnect.Spec.ImageRef.Repository = "not a correct repo"

		_, err := updater.combineImageWithDigest(t.Context(), fakeDigest)
		require.Error(t, err)
	})
}

func Test_updater_RequiresReconcile(t *testing.T) {
	currentTime := timeprovider.New().Freeze()
	fakeRegistryClient := registrymock.NewImageGetter(t)

	t.Run("initial reconcile always required", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = new(true)
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		edgeConnectTime := metav1.NewTime(currentTime.Now().Add(-time.Hour))
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = new(true)
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect(t)
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()
		edgeConnect.Spec.ImageRef = image.Ref{
			Repository: "docker.io/dynatrace/superfancynew",
		}

		assert.True(t, updater.RequiresReconcile())
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
