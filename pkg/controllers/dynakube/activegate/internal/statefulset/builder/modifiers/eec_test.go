package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEecEnabled(t *testing.T) {
	t.Run("Extensions are enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		dk.Spec.Extensions.Enabled = true

		mod := NewEecVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("Extension are disabled", func(t *testing.T) {
		dk := getBaseDynakube()
		dk.Spec.Extensions.Enabled = false

		mod := NewEecVolumeModifier(dk)

		assert.False(t, mod.Enabled())
	})
}

func TestEecModify(t *testing.T) {
	t.Run("Statefulset is successfully modified with eec volume", func(t *testing.T) {
		dk := getBaseDynakube()
		dk.Spec.Extensions.Enabled = true

		mod := NewEecVolumeModifier(dk)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
		require.Equal(t, eecVolumeName, sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
		require.Equal(t, eecMountPath, sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
		require.True(t, sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	})
}
