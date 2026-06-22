package registry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveImage(t *testing.T) {
	const testURI = "registry.example.com/dynatrace/eec:1.2.3"
	const testOverride = "custom.registry.example.com"
	noTemplate := &image.Ref{}

	t.Run("errors when neither template nor public registry is configured", func(t *testing.T) {
		uri, err := ResolveImage(t.Context(), nil, false, "", dtimage.EEC, noTemplate)
		require.Error(t, err)
		assert.Empty(t, uri)
	})

	t.Run("returns URI when public registry is enabled", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), dtimage.EEC, "").Return(&dtimage.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, true, "", dtimage.EEC, noTemplate)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("passes registryOverride to client", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), dtimage.DBExecutor, testOverride).Return(&dtimage.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, true, testOverride, dtimage.DBExecutor, noTemplate)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("propagates error from client", func(t *testing.T) {
		mockClient := imagemock.NewClient(t)
		boom := errors.New("connection refused")
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), dtimage.LogModule, "").Return(nil, boom)

		uri, err := ResolveImage(t.Context(), mockClient, true, "", dtimage.LogModule, noTemplate)
		require.ErrorIs(t, err, boom)
		assert.Empty(t, uri)
	})

	t.Run("returns template image when set, skipping API call", func(t *testing.T) {
		templateRef := &image.Ref{Repository: "my-registry/my-repo", Tag: "1.0.0"}

		uri, err := ResolveImage(t.Context(), nil, true, "", dtimage.EEC, templateRef)
		require.NoError(t, err)
		assert.Equal(t, "my-registry/my-repo:1.0.0", uri)
	})
}
