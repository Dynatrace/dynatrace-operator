package version

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry/mocks"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

func TestReconcile(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	fakeRegistryClient := &mocks.MockImageGetter{}
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	updater := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

	err := updater.Update(context.TODO())
	require.NoError(t, err)

	require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:latest@%s", fakeDigest), edgeConnect.Status.Version.ImageID)
	require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)

	// check invalid digest
	invalidImageVersion := registry.ImageVersion{Digest: "invaliddigest"}
	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(invalidImageVersion, nil)

	updater = newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, edgeConnect)

	err = updater.Update(context.TODO())
	require.NoError(t, err)

	// digest should not have been updated due to probe timestamp
	require.True(t, strings.Contains(edgeConnect.Status.Version.ImageID, fakeDigest))
}

func TestCombineImagesWithDigest(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	fakeRegistryClient := &mocks.MockImageGetter{}
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	updater := newUpdater(fake.NewClient(), nil, fakeRegistryClient, edgeConnect)

	t.Run("image and digest should be combined", func(t *testing.T) {
		combined, err := updater.combineImageWithDigest(fakeDigest)

		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:latest@%s", fakeDigest), combined)
	})

	t.Run("malformed image should fail", func(t *testing.T) {
		edgeConnect.Spec.ImageRef.Repository = "not a correct repo"

		_, err := updater.combineImageWithDigest(fakeDigest)
		require.Error(t, err)
	})
}

func TestReconcileRequired(t *testing.T) {
	currentTime := timeprovider.New().Freeze()
	mockImageGetter := &mocks.MockImageGetter{}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)

	t.Run("initial reconcile always required", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, mockImageGetter, edgeConnect)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, mockImageGetter, edgeConnect)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, mockImageGetter, edgeConnect)

		edgeConnectTime := metav1.NewTime(currentTime.Now().Add(-time.Hour))
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newUpdater(fake.NewClient(), currentTime, mockImageGetter, edgeConnect)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()
		edgeConnect.Spec.ImageRef = edgeconnectv1alpha1.ImageRefSpec{
			Repository: "docker.io/dynatrace/superfancynew",
		}

		assert.True(t, updater.RequiresReconcile())
	})
}

func createBasicEdgeConnect() *edgeconnectv1alpha1.EdgeConnect {
	return &edgeconnectv1alpha1.EdgeConnect{
		Spec: edgeconnectv1alpha1.EdgeConnectSpec{
			ApiServer: "superfancy.dev.apps.dynatracelabs.com",
		},
		Status: edgeconnectv1alpha1.EdgeConnectStatus{},
	}
}
