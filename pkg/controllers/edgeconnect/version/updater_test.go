package version

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/oci/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("default image => registry used", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

		updater := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

		err := updater.Update(ctx)
		require.NoError(t, err)

		require.Equal(t, "docker.io/dynatrace/edgeconnect:latest@"+fakeDigest, edgeConnect.Status.Version.ImageID)
		require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)

		// check invalid digest
		invalidImageVersion := registry.ImageVersion{Digest: "invaliddigest"}
		fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(invalidImageVersion, nil)

		updater = newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

		err = updater.Update(ctx)
		require.NoError(t, err)

		// digest should not have been updated due to probe timestamp
		require.True(t, strings.Contains(edgeConnect.Status.Version.ImageID, fakeDigest))
	})

	t.Run("custom tag used => registry still used", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
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
		edgeConnect := createBasicEdgeConnect()
		customRegistry := "best.registry.io/dynatrace/edgeconnect"
		edgeConnect.Spec.ImageRef.Repository = customRegistry

		updater := newUpdater(fake.NewClient(), timeprovider.New(), nil, edgeConnect)

		err := updater.Update(ctx)
		require.NoError(t, err)

		require.Equal(t, customRegistry+":latest", edgeConnect.Status.Version.ImageID)
		require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)
	})
}

func TestCombineImagesWithDigest(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	fakeRegistryClient := registrymock.NewImageGetter(t)

	updater := newUpdater(fake.NewClient(), nil, fakeRegistryClient, edgeConnect)

	t.Run("image and digest should be combined", func(t *testing.T) {
		combined, err := updater.combineImageWithDigest(fakeDigest)

		require.NoError(t, err)
		require.Equal(t, "docker.io/dynatrace/edgeconnect:latest@"+fakeDigest, combined)
	})

	t.Run("malformed image should fail", func(t *testing.T) {
		edgeConnect.Spec.ImageRef.Repository = "not a correct repo"

		_, err := updater.combineImageWithDigest(fakeDigest)
		require.Error(t, err)
	})
}

func TestReconcileRequired(t *testing.T) {
	currentTime := timeprovider.New().Freeze()
	fakeRegistryClient := registrymock.NewImageGetter(t)

	t.Run("initial reconcile always required", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, edgeConnect)

		edgeConnectTime := metav1.NewTime(currentTime.Now().Add(-time.Hour))
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
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

func createBasicEdgeConnect() *edgeconnect.EdgeConnect {
	return &edgeconnect.EdgeConnect{
		Spec: edgeconnect.EdgeConnectSpec{
			ApiServer: "superfancy.dev.apps.dynatracelabs.com",
		},
		Status: edgeconnect.EdgeConnectStatus{},
	}
}
