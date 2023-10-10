package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinityNodeRequirement(t *testing.T) {
	assert.Equal(t, AffinityNodeRequirementForSupportedArches(), affinityNodeRequirementsForArches(amd64, arm64, ppc64le))
	assert.Contains(t, AffinityNodeRequirementForSupportedArches(), linuxRequirement())
}

func TestTolerationsForAllArches(t *testing.T) {
	assert.Equal(t, TolerationForSupportedArches(), tolerationsForArches(amd64, arm64, ppc64le))
	assert.Contains(t, TolerationForSupportedArches(), armToleration())
	assert.Contains(t, TolerationForSupportedArches(), amdToleration())
}

func armToleration() corev1.Toleration {
	return corev1.Toleration{
		Key:      kubernetesArch,
		Operator: corev1.TolerationOpEqual,
		Value:    arm64,
		Effect:   corev1.TaintEffectNoSchedule,
	}
}

func amdToleration() corev1.Toleration {
	return corev1.Toleration{
		Key:      kubernetesArch,
		Operator: corev1.TolerationOpEqual,
		Value:    amd64,
		Effect:   corev1.TaintEffectNoSchedule,
	}
}

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{linux},
	}
}
