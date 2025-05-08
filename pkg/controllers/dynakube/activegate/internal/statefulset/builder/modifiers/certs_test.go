package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTlsSecretName = "test-tls-secret"

func setCertUsage(dk *dynakube.DynaKube, isUsed bool) {
	if isUsed {
		dk.Spec.ActiveGate.TlsSecretName = testTlsSecretName
	} else {
		dk.Spec.ActiveGate.TlsSecretName = ""
	}
}

func disableAutomaticAGCertificate(dk *dynakube.DynaKube) {
	dk.Annotations[exp.AGAutomaticTLSCertificateKey] = "false"
}

func TestCertEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		disableAutomaticAGCertificate(&dk)
		enableKubeMonCapability(&dk)
		setCertUsage(&dk, true)

		mod := NewCertificatesModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		disableAutomaticAGCertificate(&dk)
		enableKubeMonCapability(&dk)
		setCertUsage(&dk, false)

		mod := NewCertificatesModifier(dk)

		assert.False(t, mod.Enabled())
	})

	t.Run("true, AG cert enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setCertUsage(&dk, true)

		mod := NewCertificatesModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false, AG cert enabled", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setCertUsage(&dk, false)

		mod := NewCertificatesModifier(dk)

		assert.True(t, mod.Enabled())
	})
}

func TestCertModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setCertUsage(&dk, true)
		mod := NewCertificatesModifier(dk)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
