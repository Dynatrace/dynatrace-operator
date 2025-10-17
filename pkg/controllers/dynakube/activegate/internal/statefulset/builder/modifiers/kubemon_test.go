package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setKubernetesMonitoringUsage(dk *dynakube.DynaKube, isUsed bool) {
	if isUsed {
		enableKubeMonCapability(dk)
	}
}

func TestKubernetesMonitoringEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dk := getBaseDynakube()
		setKubernetesMonitoringUsage(&dk, true)
		multiCapability := capability.NewMultiCapability(&dk)

		mod := NewKubernetesMonitoringModifier(dk, multiCapability)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dk := getBaseDynakube()
		setKubernetesMonitoringUsage(&dk, false)
		multiCapability := capability.NewMultiCapability(&dk)

		mod := NewKubernetesMonitoringModifier(dk, multiCapability)

		assert.False(t, mod.Enabled())
	})
}

func TestKubernetesMonitoringModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		setKubernetesMonitoringUsage(&dk, true)
		multiCapability := capability.NewMultiCapability(&dk)
		mod := NewKubernetesMonitoringModifier(dk, multiCapability)
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

		require.NotNil(t, sts.Spec.Template.Spec.AutomountServiceAccountToken)
		assert.True(t, *sts.Spec.Template.Spec.AutomountServiceAccountToken)
	})
	t.Run("successfully modified with readonly feature flag", func(t *testing.T) {
		dk := getBaseDynakube()
		setKubernetesMonitoringUsage(&dk, true)
		multiCapability := capability.NewMultiCapability(&dk)
		mod := NewKubernetesMonitoringModifier(dk, multiCapability)
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
