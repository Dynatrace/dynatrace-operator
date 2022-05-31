package pod_mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallInitContainerBase(t *testing.T) {
	t.Run("should create the init container", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		webhookImage := "test-image"
		initContainer := createInstallInitContainerBase(webhookImage, pod, dynakube)
		require.NotNil(t, initContainer)
		assert.Equal(t, initContainer.Image, webhookImage)
		assert.Equal(t, initContainer.Resources, testResourceRequirements)
		assert.Equal(t, initContainer.SecurityContext, testSecurityContext)
	})
}
