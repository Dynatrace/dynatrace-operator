package registry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResolveImage(t *testing.T) {
	const testURI = "registry.example.com/dynatrace/eec:1.2.3"
	const testOverride = "custom.registry.example.com"

	t.Run("returns empty when public registry is not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		uri, err := ResolveImage(t.Context(), nil, dk, image.EEC)
		require.NoError(t, err)
		assert.Empty(t, uri)
	})

	t.Run("returns URI when platform token is present", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				APIToken: dynakube.APITokenStatus{Platform: new(true)},
			},
		}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(&image.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, dk, image.EEC)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("returns URI when use-public-registry annotation is set", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{exp.UsePublicRegistryKey: "true"},
			},
		}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(&image.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, dk, image.LogModule)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("passes PublicRegistryOverride to client", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{exp.UsePublicRegistryKey: "true"},
			},
			Spec: dynakube.DynaKubeSpec{PublicRegistryOverride: testOverride},
		}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.DBExecutor, testOverride).Return(&image.Info{URI: testURI}, nil)

		uri, err := ResolveImage(t.Context(), mockClient, dk, image.DBExecutor)
		require.NoError(t, err)
		assert.Equal(t, testURI, uri)
	})

	t.Run("propagates error from client", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				APIToken: dynakube.APITokenStatus{Platform: new(true)},
			},
		}
		mockClient := imagemock.NewClient(t)
		boom := errors.New("connection refused")
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(nil, boom)

		uri, err := ResolveImage(t.Context(), mockClient, dk, image.EEC)
		require.ErrorIs(t, err, boom)
		assert.Empty(t, uri)
	})
}
