package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProxyName = "test-proxy"

func setProxyUsage(dk *dynakube.DynaKube, isUsed bool) {
	dk.Spec.Proxy = &dynakube.DynaKubeProxy{}
	if isUsed {
		dk.Spec.Proxy.ValueFrom = testProxyName
	} else {
		dk.Spec.Proxy.ValueFrom = ""
	}
}

func TestProxyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setProxyUsage(&dynakube, true)

		mod := NewProxyModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setProxyUsage(&dynakube, false)

		mod := NewProxyModifier(dynakube)

		assert.False(t, mod.Enabled())
	})
}

func TestProxyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setProxyUsage(&dynakube, true)
		mod := NewProxyModifier(dynakube)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
