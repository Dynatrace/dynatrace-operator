package modifiers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSLVolumeEnabled(t *testing.T) {
	t.Run("true - TlsSecretName", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		dynakube.Spec.ActiveGate.TlsSecretName = testTlsSecretName

		mod := NewSSLVolumeModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("true - TrustedCAs", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		dynakube.Spec.TrustedCAs = testTlsSecretName

		mod := NewSSLVolumeModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)

		mod := NewSSLVolumeModifier(dynakube)

		assert.False(t, mod.Enabled())
	})
}

func TestSSLVolumeModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		dynakube.Spec.ActiveGate.TlsSecretName = testTlsSecretName

		mod := NewSSLVolumeModifier(dynakube)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
