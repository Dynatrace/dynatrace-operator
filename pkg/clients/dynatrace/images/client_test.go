package images

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ComponentLatestImageURI(t *testing.T) {
	setupClient := func(t *testing.T, err error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			Execute(new([]ComponentResponse)).
			Run(func(model any) {
				if err == nil {
					resp := model.(*[]ComponentResponse)
					*resp = []ComponentResponse{
						{Type: OneAgent, ImageURI: "image:tag@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"},
					}
				}
			}).
			Return(err).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), containerImagesPath).Return(req).Once()

		return NewClient(client, "")
	}

	t.Run("found", func(t *testing.T) {
		client := setupClient(t, nil)
		imageURI, err := client.ComponentLatestImageURI(t.Context(), OneAgent)
		require.NoError(t, err)
		assert.Equal(t, "image:tag@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80", imageURI)
	})

	t.Run("not found", func(t *testing.T) {
		client := setupClient(t, nil)
		_, err := client.ComponentLatestImageURI(t.Context(), "aasddasd")
		require.Error(t, err)
	})

	t.Run("api error 404", func(t *testing.T) {
		client := setupClient(t, &core.HTTPError{StatusCode: 404, Message: "nope"})
		_, err := client.ComponentLatestImageURI(t.Context(), ActiveGate)
		require.True(t, core.IsNotFound(err))
		assert.EqualError(t, err, "get latest activegate image: nope")
	})
}

func Test_test(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		imageURI := "478983378254.dkr.ecr.us-east-1.amazonaws.com/dynatrace/dynatrace-oneagent:1.336.0@sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"
		ref, err := name.ParseReference(imageURI, name.StrictValidation)
		require.NoError(t, err)
		assert.Equal(t, ref.Context().RegistryStr(), "478983378254.dkr.ecr.us-east-1.amazonaws.com")
	})
}
