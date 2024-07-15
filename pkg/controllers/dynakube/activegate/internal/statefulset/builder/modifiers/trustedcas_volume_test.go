package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedCAsVolumeEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.TrustedCAs = testTlsSecretName

		mod := NewTrustedCAsVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false - TlsSecretName", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.ActiveGate.TlsSecretName = testTlsSecretName

		mod := NewTrustedCAsVolumeModifier(dk)

		assert.False(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)

		mod := NewTrustedCAsVolumeModifier(dk)

		assert.False(t, mod.Enabled())
	})
}

func TestTrustedCAsVolumeModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.TrustedCAs = testTlsSecretName

		mod := NewTrustedCAsVolumeModifier(dk)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
