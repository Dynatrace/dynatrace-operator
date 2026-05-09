package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentPropertiesModifierEnabled(t *testing.T) {
	t.Run("enabled when resource attributes are set", func(t *testing.T) {
		dk := getBaseDynakube()
		dk.Spec.ResourceAttributes = map[string]string{"key": "value"}
		mod := NewDeploymentPropertiesModifier(dk)
		assert.True(t, mod.Enabled())
	})

	t.Run("disabled without resource attributes", func(t *testing.T) {
		dk := getBaseDynakube()
		mod := NewDeploymentPropertiesModifier(dk)
		assert.False(t, mod.Enabled())
	})
}

func TestDeploymentPropertiesModifierModify(t *testing.T) {
	t.Run("adds volume and volumeMount with correct paths", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.ResourceAttributes = map[string]string{"key": "value"}

		mod := NewDeploymentPropertiesModifier(dk)
		b := createBuilderForTesting()

		sts, _ := b.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("volume references correct configmap name", func(t *testing.T) {
		dk := getBaseDynakube()
		mod := NewDeploymentPropertiesModifier(dk)

		volumes := mod.getVolumes()
		require.Len(t, volumes, 1)
		assert.Equal(t, consts.DeploymentPropertiesVolumeName, volumes[0].Name)
		assert.Equal(t, dk.ActiveGate().GetDeploymentPropertiesConfigMapName(), volumes[0].ConfigMap.Name)
		assert.Equal(t, testDynakubeName+activegate.DeploymentPropertiesConfigMapSuffix, volumes[0].ConfigMap.Name)
	})

	t.Run("volumeMount has correct mountPath and subPath", func(t *testing.T) {
		dk := getBaseDynakube()
		mod := NewDeploymentPropertiesModifier(dk)

		mounts := mod.getVolumeMounts()
		require.Len(t, mounts, 1)
		assert.Equal(t, getMountPath(), mounts[0].MountPath)
		assert.Equal(t, consts.DeploymentPropertiesFileName, mounts[0].SubPath)
		assert.True(t, mounts[0].ReadOnly)
	})
}
