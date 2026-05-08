package images

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ComponentLatestImageInfo(t *testing.T) {
	const expectedTag = "tag"
	const expectedImageURI = "image:tag@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"

	setupClient := func(t *testing.T, apiErr error, params map[string]string, imageURI string) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithQueryParams(params).Return(req).Once()
		req.EXPECT().
			Execute(new(containerImagesResponse)).
			Run(func(model any) {
				if apiErr == nil {
					resp := model.(*containerImagesResponse)
					*resp = containerImagesResponse{
						Components: []componentResponse{
							{Type: OneAgent, ImageURI: imageURI},
						},
					}
				}
			}).
			Return(apiErr).Once()
		client := coremock.NewClient(t)
		client.EXPECT().GET(t.Context(), containerImagesPath).Return(req).Once()

		return NewClient(client)
	}

	t.Run("found", func(t *testing.T) {
		client := setupClient(t, nil, map[string]string{}, expectedImageURI)
		imageInfo, err := client.ComponentLatestImageInfo(t.Context(), OneAgent, "")
		require.NoError(t, err)
		assert.Equal(t, expectedTag, imageInfo.Tag)
	})

	t.Run("not found", func(t *testing.T) {
		client := setupClient(t, nil, map[string]string{}, expectedImageURI)
		_, err := client.ComponentLatestImageInfo(t.Context(), "aasddasd", "")
		require.Error(t, err)
	})

	t.Run("api error 404", func(t *testing.T) {
		client := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"}, map[string]string{}, expectedImageURI)
		_, err := client.ComponentLatestImageInfo(t.Context(), ActiveGate, "")
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, "get latest activegate image: nope")
	})

	t.Run("registry override passed as query param", func(t *testing.T) {
		const customRegistry = "my.custom.registry.com"
		const imageURI = customRegistry + "/image:tag"
		client := setupClient(t, nil, map[string]string{"registry": customRegistry}, imageURI)
		imageInfo, err := client.ComponentLatestImageInfo(t.Context(), OneAgent, customRegistry)
		require.NoError(t, err)
		assert.Equal(t, expectedTag, imageInfo.Tag)
	})

	t.Run("registry mismatch returns error", func(t *testing.T) {
		const requestedRegistry = "my.custom.registry.com"
		const imageURI = "other.registry.com/dynatrace/oneagent:tag"
		client := setupClient(t, nil, map[string]string{"registry": requestedRegistry}, imageURI)
		_, err := client.ComponentLatestImageInfo(t.Context(), OneAgent, requestedRegistry)
		require.Error(t, err)
		assert.EqualError(t, err, `image registry "other.registry.com" does not match requested registry "my.custom.registry.com"`)
	})
}

func Test_parseImageInfo(t *testing.T) {
	expectedRegistry := "some.amazonaws.com"
	expectedTag := "1.336.0"
	testDigest := "sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"
	baseImage := expectedRegistry + "/dynatrace/some-image"

	t.Run("tag and digest", func(t *testing.T) {
		imageURI := baseImage + ":" + expectedTag + "@" + testDigest
		info, err := parseImageInfo(imageURI)
		require.NoError(t, err)
		assert.Equal(t, expectedTag, info.Tag)
		assert.Equal(t, expectedRegistry, info.Registry)
	})

	t.Run("tag only", func(t *testing.T) {
		imageURI := baseImage + ":" + expectedTag
		info, err := parseImageInfo(imageURI)
		require.NoError(t, err)
		assert.Equal(t, expectedTag, info.Tag)
		assert.Equal(t, expectedRegistry, info.Registry)
	})

	t.Run("digest only", func(t *testing.T) {
		imageURI := baseImage + "@" + testDigest
		info, err := parseImageInfo(imageURI)
		require.NoError(t, err)
		assert.Empty(t, info.Tag)
		assert.Equal(t, expectedRegistry, info.Registry)
	})
}
