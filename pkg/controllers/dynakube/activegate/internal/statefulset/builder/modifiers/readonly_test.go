package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOnlyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)

		mod := NewReadOnlyModifier(dk)

		assert.True(t, mod.Enabled())
	})
}

func TestReadOnlyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		mod := NewReadOnlyModifier(dk)
		builder := createBuilderForTesting()
		expectedVolumes := mod.getVolumes()
		expectedVolumeMounts := mod.getVolumeMounts()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, expectedVolumes, sts.Spec.Template.Spec.Volumes)
		isSubset(t, expectedVolumeMounts, sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
