package modifiers

import (
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setServicePortUsage(dynakube *dynatracev1beta2.DynaKube, isUsed bool) {
	if isUsed {
		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1beta2.MetricsIngestCapability.DisplayName)
	}
}

func TestServicePortEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, false)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())

		assert.False(t, mod.Enabled())
	})
}

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())
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
