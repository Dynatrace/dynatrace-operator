package pod_mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallInitContainerBaseWithDefaultUserAndGroup(t *testing.T) {
	t.Run("should create the init container with default user and group", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPodWithNoSecurityContext()
		webhookImage := "test-image"
		clusterID := "id"
		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)
		require.NotNil(t, initContainer)
		assert.Equal(t, initContainer.Image, webhookImage)
		assert.Equal(t, initContainer.Resources, testResourceRequirements)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initContainer.SecurityContext.Privileged)
		assert.True(t, *initContainer.SecurityContext.ReadOnlyRootFilesystem)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, int64(1001))
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, int64(1001))
	})
}

func TestCreateInstallInitContainerBaseWithSetUserAndGroup(t *testing.T) {
	t.Run("should create the init container with set user and group", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"
		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)
		require.NotNil(t, initContainer)
		assert.Equal(t, initContainer.Image, webhookImage)
		assert.Equal(t, initContainer.Resources, testResourceRequirements)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initContainer.SecurityContext.Privileged)
		assert.True(t, *initContainer.SecurityContext.ReadOnlyRootFilesystem)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, int64(420))
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, int64(420))
	})
}
