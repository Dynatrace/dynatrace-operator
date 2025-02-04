package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
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
