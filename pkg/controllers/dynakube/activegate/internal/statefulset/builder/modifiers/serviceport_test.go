package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setServicePortUsage(dk *dynakube.DynaKube, isUsed bool) {
	if isUsed {
		dk.Spec.ActiveGate.Capabilities = append(dk.Spec.ActiveGate.Capabilities, activegate.MetricsIngestCapability.DisplayName)
	}
}

func TestServicePortEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		setServicePortUsage(&dk, true)
		multiCapability := capability.NewMultiCapability(&dk)

		mod := NewServicePortModifier(dk, multiCapability, prioritymap.New())

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		setServicePortUsage(&dk, false)
		multiCapability := capability.NewMultiCapability(&dk)

		mod := NewServicePortModifier(dk, multiCapability, prioritymap.New())

		assert.False(t, mod.Enabled())
	})
}

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		setServicePortUsage(&dk, true)
		multiCapability := capability.NewMultiCapability(&dk)
		mod := NewServicePortModifier(dk, multiCapability, prioritymap.New())
		builder := createBuilderForTesting()
		expectedPorts := mod.getPorts()
		expectedEnv := mod.getEnvs()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		container := sts.Spec.Template.Spec.Containers[0]
		isSubset(t, expectedPorts, container.Ports)
		isSubset(t, expectedEnv, container.Env)
		assert.Equal(t, consts.HttpsServicePortName, container.ReadinessProbe.HTTPGet.Port.StrVal)
	})
}
