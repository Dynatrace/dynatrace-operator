package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProxyName = "test-proxy"

func setProxyUsage(dk *dynakube.DynaKube, isUsed bool) {
	dk.Spec.Proxy = &value.Source{}
	if isUsed {
		dk.Spec.Proxy.ValueFrom = testProxyName
	} else {
		dk.Spec.Proxy.ValueFrom = ""
	}
}

func TestProxyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setProxyUsage(&dk, true)

		mod := NewProxyModifier(dk)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setProxyUsage(&dk, false)

		mod := NewProxyModifier(dk)

		assert.False(t, mod.Enabled())
	})
}

func TestProxyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		setProxyUsage(&dk, true)
		mod := NewProxyModifier(dk)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
