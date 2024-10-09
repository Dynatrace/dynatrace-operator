package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCustomPropertyValue = "testing-property"

func setCustomPropertyUsage(capability capability.Capability, isUsed bool) {
	if isUsed {
		capability.Properties().CustomProperties = &value.Source{
			Value: testCustomPropertyValue,
		}
	} else {
		capability.Properties().CustomProperties = nil
	}
}

func TestCustomPropertyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		multiCapability := capability.NewMultiCapability(&dk)
		setCustomPropertyUsage(multiCapability, true)

		mod := NewCustomPropertiesModifier(dk, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("true with NeedsCustomNoProxy", func(t *testing.T) {
		dk := getBaseDynakube()
		dk.Spec.Proxy = &value.Source{
			Value: "test",
		}
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNoProxy: "test.example.com",
		}

		enableKubeMonCapability(&dk)
		multiCapability := capability.NewMultiCapability(&dk)
		setCustomPropertyUsage(multiCapability, false)

		mod := NewCustomPropertiesModifier(dk, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		multiCapability := capability.NewMultiCapability(&dk)
		setCustomPropertyUsage(multiCapability, false)

		mod := NewCustomPropertiesModifier(dk, multiCapability)

		assert.False(t, mod.Enabled())
	})
}

func TestCustomPropertyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		multiCapability := capability.NewMultiCapability(&dk)
		setCustomPropertyUsage(multiCapability, true)
		mod := NewCustomPropertiesModifier(dk, multiCapability)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
