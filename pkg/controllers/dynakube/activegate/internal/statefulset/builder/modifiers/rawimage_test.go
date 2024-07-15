package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawImageEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)

		mod := NewRawImageModifier(dk, prioritymap.New())

		assert.True(t, mod.Enabled())
	})
}

func TestRawImageModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		mod := NewRawImageModifier(dk, prioritymap.New())
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
		isSubset(t, mod.getEnvs(), sts.Spec.Template.Spec.Containers[0].Env)
	})
}
