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
	"k8s.io/utils/ptr"
)

const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

func testBasicEdgeConnect() *edgeconnect.EdgeConnect {
	return &edgeconnect.EdgeConnect{
		Spec: edgeconnect.EdgeConnectSpec{
			APIServer: "superfancy.dev.apps.dynatracelabs.com",
		},
		Status: edgeconnect.EdgeConnectStatus{},
	}
}

func Test_updater_Update(t *testing.T) {
	t.Run("default image => registry used", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.EXPECT().GetImageVersion(mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Once()

		u := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, ec)

		err := u.Update(t.Context())
		require.NoError(t, err)

		require.Equal(t, "docker.io/dynatrace/edgeconnect:latest@"+fakeDigest, ec.Status.Version.ImageID)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("custom tag used => registry still used", func(t *testing.T) {
		const customTag = "1.2.3"

		ec := testBasicEdgeConnect()
		ec.Spec.ImageRef.Tag = customTag

		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.EXPECT().GetImageVersion(mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Once()

		u := newUpdater(fake.NewClient(), timeprovider.New(), fakeRegistryClient, ec)

		err := u.Update(t.Context())
		require.NoError(t, err)

		require.Equal(t, fmt.Sprintf("docker.io/dynatrace/edgeconnect:%s@%s", customTag, fakeDigest), ec.Status.Version.ImageID)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})

	t.Run("custom registry used => registry NOT used", func(t *testing.T) {
		const customRegistry = "best.registry.io/dynatrace/edgeconnect"

		ec := testBasicEdgeConnect()
		ec.Spec.ImageRef.Repository = customRegistry

		u := newUpdater(fake.NewClient(), timeprovider.New(), nil, ec)

		err := u.Update(t.Context())
		require.NoError(t, err)

		require.Equal(t, customRegistry+":latest", ec.Status.Version.ImageID)
		require.NotNil(t, ec.Status.Version.LastProbeTimestamp)
	})
}

func Test_updater_combineImageWithDigest(t *testing.T) {
	t.Run("image and digest should be combined", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), nil, fakeRegistryClient, ec)

		combined, err := u.combineImageWithDigest(fakeDigest)

		require.NoError(t, err)
		require.Equal(t, "docker.io/dynatrace/edgeconnect:latest@"+fakeDigest, combined)
	})

	t.Run("malformed image should fail", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		ec.Spec.ImageRef.Repository = "not a correct repo"

		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), nil, fakeRegistryClient, ec)

		_, err := u.combineImageWithDigest(fakeDigest)
		require.Error(t, err)
	})
}

func Test_updater_RequiresReconcile(t *testing.T) {
	currentTime := timeprovider.New().Freeze()

	t.Run("initial reconcile always required", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, ec)

		assert.True(t, u.RequiresReconcile(), "initial reconcile always required")
	})

	t.Run("only reconcile every threshold minutes", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, ec)

		ecTime := metav1.Now()
		ec.Status.Version.LastProbeTimestamp = &ecTime
		ec.Spec.AutoUpdate = ptr.To(true)
		ec.Status.Version.ImageID = ec.Image()

		assert.False(t, u.RequiresReconcile())
	})

	t.Run("reconcile as auto update was enabled and time is up", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, ec)

		ecTime := metav1.NewTime(currentTime.Now().Add(-time.Hour))
		ec.Status.Version.LastProbeTimestamp = &ecTime
		ec.Spec.AutoUpdate = ptr.To(true)
		ec.Status.Version.ImageID = ec.Image()

		assert.True(t, u.RequiresReconcile())
	})

	t.Run("reconcile if image field changed", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		u := newUpdater(fake.NewClient(), currentTime, fakeRegistryClient, ec)

		ecTime := metav1.Now()
		ec.Status.Version.LastProbeTimestamp = &ecTime
		ec.Status.Version.ImageID = ec.Image()
		ec.Spec.ImageRef = image.Ref{
			Repository: "docker.io/dynatrace/superfancynew",
		}

		assert.True(t, u.RequiresReconcile())
	})
}
