package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setKubernetesMonitoringUsage(dynakube *dynatracev1beta1.DynaKube, isUsed bool) {
	if isUsed {
		enableKubeMonCapability(dynakube)
	}
}

func TestKubernetesMonitoringEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setKubernetesMonitoringUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewKubernetesMonitoringModifier(dynakube, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setKubernetesMonitoringUsage(&dynakube, false)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewKubernetesMonitoringModifier(dynakube, multiCapability)

		assert.False(t, mod.Enabled())
	})
}

func TestKubernetesMonitoringModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setKubernetesMonitoringUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewKubernetesMonitoringModifier(dynakube, multiCapability)
		builder := createBuilderForTesting()
		expectedVolumes := mod.getVolumes()
		expectedIniContainers := mod.getInitContainers()
		expectedVolumeMounts := mod.getVolumeMounts()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		container := sts.Spec.Template.Spec.Containers[0]
		isSubset(t, expectedVolumes, sts.Spec.Template.Spec.Volumes)
		isSubset(t, expectedVolumeMounts, container.VolumeMounts)
		isSubset(t, expectedIniContainers, sts.Spec.Template.Spec.InitContainers)
	})
	t.Run("successfully modified with readonly feature flag", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setKubernetesMonitoringUsage(&dynakube, true)
		setReadOnlyUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewKubernetesMonitoringModifier(dynakube, multiCapability)
		builder := createBuilderForTesting()
		expectedVolumes := mod.getVolumes()
		expectedIniContainers := mod.getInitContainers()
		expectedVolumeMounts := mod.getVolumeMounts()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		container := sts.Spec.Template.Spec.Containers[0]
		isSubset(t, expectedVolumes, sts.Spec.Template.Spec.Volumes)
		isSubset(t, expectedVolumeMounts, container.VolumeMounts)
		isSubset(t, expectedIniContainers, sts.Spec.Template.Spec.InitContainers)
	})
}
