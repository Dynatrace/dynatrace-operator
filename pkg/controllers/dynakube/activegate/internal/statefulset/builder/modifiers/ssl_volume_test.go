package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSLVolumeEnabled(t *testing.T) {
	t.Run("true - TLSSecretName", func(t *testing.T) {
		dk := getBaseDynakube()
		disableAutomaticAGCertificate(&dk)
		enableKubeMonCapability(&dk)
		dk.Spec.ActiveGate.TLSSecretName = testTLSSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("true - TrustedCAs", func(t *testing.T) {
		dk := getBaseDynakube()
		disableAutomaticAGCertificate(&dk)
		enableKubeMonCapability(&dk)
		dk.Spec.TrustedCAs = testTLSSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		disableAutomaticAGCertificate(&dk)
		enableKubeMonCapability(&dk)

		mod := NewSSLVolumeModifier(dk)

		assert.False(t, mod.Enabled())
	})

	t.Run("true - TLSSecretName, AG cert enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.ActiveGate.TLSSecretName = testTLSSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("true - TrustedCAs, AG cert enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		dk.Spec.TrustedCAs = testTLSSecretName

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false, AG cert enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)

		mod := NewSSLVolumeModifier(dk)

		assert.True(t, mod.Enabled())
	})
}

func TestSSLVolumeModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dl := getBaseDynakube()
		enableKubeMonCapability(&dl)
		dl.Spec.ActiveGate.TLSSecretName = testTLSSecretName

		mod := NewSSLVolumeModifier(dl)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
