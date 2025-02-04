package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSLVolumeEnabled(t *testing.T) {
	t.Run("true - TlsSecretName", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.ActiveGate.TlsSecretName = testTlsSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("true - TrustedCAs", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.TrustedCAs = testTlsSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)

		mod := NewSSLVolumeModifier(dk)

		assert.False(t, mod.Enabled())
	})
}

func TestSSLVolumeModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dl := getBaseDynakube()
		enableKubeMonCapability(&dl)
		dl.Spec.ActiveGate.TlsSecretName = testTlsSecretName

		mod := NewSSLVolumeModifier(dl)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
