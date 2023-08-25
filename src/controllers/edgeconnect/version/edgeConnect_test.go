package version

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

type registryClientMock struct {
	digest string
}

func (client *registryClientMock) GetImageVersion(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (registry.ImageVersion, error) {
	if imageName == "docker.io/dynatrace/edgeconnect:latest" {
		return registry.ImageVersion{
			Digest: fakeDigest,
		}, nil
	}

	return registry.ImageVersion{}, fmt.Errorf("This should not happen")
}

func TestReconcile(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), timeprovider.New())

	fakeRegistryClient := registryClientMock{
		digest: fakeDigest,
	}
	updater.registryClient = &fakeRegistryClient

	err := updater.Update(context.TODO())
	require.NoError(t, err)

	require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:latest@%s", fakeDigest), edgeConnect.Status.Version.ImageID)
	require.NotNil(t, edgeConnect.Status.Version.LastProbeTimestamp)

	fakeRegistryClient.digest = "invaliddigest"

	err = updater.Update(context.TODO())
	require.NoError(t, err)

	// digest should not have been updated due to probe timestamp
	require.True(t, strings.Contains(edgeConnect.Status.Version.ImageID, fakeDigest))
}

func TestCombineImagesWithDigest(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), nil)

	t.Run("image and digest should be combined", func(t *testing.T) {
		combined, err := updater.combineImageWithDigest(digest.Digest(fakeDigest))

		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:latest@%s", fakeDigest), combined)
	})

	t.Run("malformed image should fail", func(t *testing.T) {
		edgeConnect.Spec.ImageRef.Repository = "not a correct repo"

		_, err := updater.combineImageWithDigest(digest.Digest(fakeDigest))
		require.Error(t, err)
	})
}

func TestReconcileRequired(t *testing.T) {
	currentTime := timeprovider.New().Freeze()

	t.Run("initial reconcile always required", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), currentTime)

		assert.True(t, updater.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), currentTime)

		edgeConnectTime := metav1.Now()
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.False(t, updater.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), currentTime)

		edgeConnectTime := metav1.NewTime(currentTime.Now().Add(-time.Hour))
		edgeConnect.Status.Version.LastProbeTimestamp = &edgeConnectTime
		edgeConnect.Spec.AutoUpdate = true
		edgeConnect.Status.Version.ImageID = edgeConnect.Image()

		assert.True(t, updater.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		edgeConnect := createBasicEdgeConnect()
		updater := newEdgeConnectUpdater(edgeConnect, fake.NewClient(), currentTime)

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
