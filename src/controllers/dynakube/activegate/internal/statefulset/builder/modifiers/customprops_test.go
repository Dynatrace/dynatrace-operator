package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCustomPropertyValue = "testing-property"

func setCustomPropertyUsage(capability capability.Capability, isUsed bool) {
	if isUsed {
		capability.Properties().CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testCustomPropertyValue,
		}
	} else {
		capability.Properties().CustomProperties = nil
	}
}

func TestCustomPropertyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		multiCapability := capability.NewMultiCapability(&dynakube)
		setCustomPropertyUsage(multiCapability, true)

		mod := NewCustomPropertiesModifier(dynakube, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		multiCapability := capability.NewMultiCapability(&dynakube)
		setCustomPropertyUsage(multiCapability, false)

		mod := NewCustomPropertiesModifier(dynakube, multiCapability)

		assert.False(t, mod.Enabled())
	})
}

func TestCustomPropertyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		multiCapability := capability.NewMultiCapability(&dynakube)
		setCustomPropertyUsage(multiCapability, true)
		mod := NewCustomPropertiesModifier(dynakube, multiCapability)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
