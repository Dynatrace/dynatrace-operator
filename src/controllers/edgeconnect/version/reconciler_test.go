package version

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/src/oci/registry/mocks"
	"github.com/Dynatrace/dynatrace-operator/src/util/timeprovider"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewReconcile(t *testing.T) {
	edgeConnect := createBasicEdgeConnect()
	fakeRegistryClient := &mocks.MockImageGetter{}
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	reconciler := NewReconciler(fake.NewClient(), fakeRegistryClient, timeprovider.New(), edgeConnect)

	require.NotNil(t, reconciler)
	require.NoError(t, reconciler.Reconcile(context.Background()))
}
