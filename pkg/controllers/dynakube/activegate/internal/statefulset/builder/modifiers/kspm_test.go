package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setKSPMUsage(dk *dynakube.DynaKube, isUsed bool) {
	dk.Spec.Kspm.Enabled = isUsed
}

func TestKspmEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setKSPMUsage(&dk, true)

		mod := NewKspmModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false - directly disabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setKSPMUsage(&dk, false)

		mod := NewKspmModifier(dk)

		assert.False(t, mod.Enabled())
	})

	t.Run("false - dependency(kubemon) not enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		setKSPMUsage(&dk, true)

		mod := NewKspmModifier(dk)

		assert.False(t, mod.Enabled())
	})
}

func TestKspmModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setKSPMUsage(&dk, true)
		mod := NewKspmModifier(dk)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
