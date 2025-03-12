package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContainer(t *testing.T) {
	tenant := "test-tenant"

	t.Run("get main container", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		mainContainer := getContainer(dk, tenant)

		require.NotEmpty(t, mainContainer)

		assert.NotEmpty(t, mainContainer.Name)
		assert.NotEmpty(t, mainContainer.Image)
		assert.NotEmpty(t, mainContainer.ImagePullPolicy)
		assert.NotEmpty(t, mainContainer.VolumeMounts)
		assert.Len(t, mainContainer.VolumeMounts, expectedMountLen)
		assert.NotEmpty(t, mainContainer.Env)
		assert.Len(t, mainContainer.Env, expectedBaseEnvLen)
		assert.NotEmpty(t, mainContainer.SecurityContext)
	})

	t.Run("image-ref is respected", func(t *testing.T) {
		expectedRepo := "my-test-repo"
		expectedTag := "my-test-tag"
		dk := dynakube.DynaKube{}
		dk.KSPM().ImageRef = image.Ref{
			Repository: expectedRepo,
			Tag:        expectedTag,
		}
		mainContainer := getContainer(dk, tenant)

		require.NotEmpty(t, mainContainer)
		assert.NotEmpty(t, mainContainer.Image)
		assert.Equal(t, expectedRepo+":"+expectedTag, mainContainer.Image)
	})
}
