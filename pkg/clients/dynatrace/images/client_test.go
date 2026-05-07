package images

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ComponentLatestImageURI(t *testing.T) {
	setupClient := func(t *testing.T, err error) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithQueryParams(map[string]string{}).Return(req).Once()
		req.EXPECT().
			Execute(new(containerImagesResponse)).
			Run(func(model any) {
				if err == nil {
					resp := model.(*containerImagesResponse)
					*resp = containerImagesResponse{
						Components: []componentResponse{
							{Type: OneAgent, ImageURI: "image:tag@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"},
						},
					}
				}
			}).
			Return(err).Once()
		client := coremock.NewClient(t)
		client.EXPECT().GET(t.Context(), containerImagesPath).Return(req).Once()

		return NewClient(client)
	}

	t.Run("found", func(t *testing.T) {
		client := setupClient(t, nil)
		imageInfo, err := client.ComponentLatestImageInfo(t.Context(), OneAgent, "")
		require.NoError(t, err)
		assert.Equal(t, "tag", imageInfo.Tag)
		assert.Equal(t, "sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80", string(imageInfo.Digest))
	})

	t.Run("not found", func(t *testing.T) {
		client := setupClient(t, nil)
		_, err := client.ComponentLatestImageInfo(t.Context(), "aasddasd", "")
		require.Error(t, err)
	})

	t.Run("api error 404", func(t *testing.T) {
		client := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"})
		_, err := client.ComponentLatestImageInfo(t.Context(), ActiveGate, "")
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, "get latest activegate image: nope")
	})
}

func Test_parseImageInfo(t *testing.T) {
	t.Run("tag and digest", func(t *testing.T) {
		imageURI := "478983378254.dkr.ecr.us-east-1.amazonaws.com/dynatrace/dynatrace-oneagent:1.336.0@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"
		info, err := parseImageInfo(imageURI)
		require.NoError(t, err)
		assert.Equal(t, "1.336.0", info.Tag)
		assert.Equal(t, "sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80", string(info.Digest))
		assert.Equal(t, "478983378254.dkr.ecr.us-east-1.amazonaws.com", info.Registry)
	})
}
