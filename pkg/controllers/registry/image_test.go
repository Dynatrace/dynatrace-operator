package registry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveImage(t *testing.T) {
	const testURI = "registry.example.com/dynatrace/eec:1.2.3"
	const testOverride = "custom.registry.example.com"

	t.Run("returns URI from public registry", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(&image.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, "", image.EEC)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("passes registryOverride to client", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.DBExecutor, testOverride).Return(&image.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, testOverride, image.DBExecutor)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("propagates error from client", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		boom := errors.New("connection refused")
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, boom)

		uri, err := ResolveImage(t.Context(), mockClient, "", image.LogModule)
		require.ErrorIs(t, err, boom)
		assert.Empty(t, uri)
	})
}
