package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTlsSecretName = "test-tls-secret"

func setCertUsage(dynakube *dynatracev1beta1.DynaKube, isUsed bool) {
	if isUsed {
		dynakube.Spec.ActiveGate.TlsSecretName = testTlsSecretName
	} else {
		dynakube.Spec.ActiveGate.TlsSecretName = ""
	}
}

func TestCertEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setCertUsage(&dynakube, true)

		mod := NewCertificatesModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setCertUsage(&dynakube, false)

		mod := NewCertificatesModifier(dynakube)

		assert.False(t, mod.Enabled())
	})
}

func TestCertModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setCertUsage(&dynakube, true)
		mod := NewCertificatesModifier(dynakube)
		builder := createBuilderForTesting()

		sts := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
