package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/oci/registry"
	"github.com/stretchr/testify/require"
)

func TestNewReconciler(t *testing.T) {
	t.Run("creates reconciler successfully", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)

		reconciler := NewReconciler(fake.NewClient(), fakeRegistryClient, timeprovider.New(), ec)

		require.NotNil(t, reconciler)
	})
}

func Test_Reconciler_Reconcile(t *testing.T) {
	t.Run("reconcile succeeds", func(t *testing.T) {
		ec := testBasicEdgeConnect()
		fakeRegistryClient := registrymock.NewImageGetter(t)
		fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
		fakeRegistryClient.EXPECT().GetImageVersion(anyCtx, ec.Image()).Return(fakeImageVersion, nil).Once()

		reconciler := NewReconciler(fake.NewClient(), fakeRegistryClient, timeprovider.New(), ec)

		require.NoError(t, reconciler.Reconcile(t.Context()))
	})
}
