package pod_mutator

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallInitContainerBase(t *testing.T) {
	t.Run("should create the init container", func(t *testing.T) {
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
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, int64(1000))
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, int64(1000))
	})
}

func TestCreateInstallInitContainerBaseReadOnlyDisabled(t *testing.T) {
	t.Run("should create the init container (read only disabled)", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "false"}
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
