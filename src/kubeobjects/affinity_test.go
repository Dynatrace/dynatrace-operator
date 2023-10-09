package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinityNodeRequirement(t *testing.T) {
	assert.Equal(t, AffinityNodeRequirementForAllArches(), affinityNodeRequirementsForArches(amd64, arm64, ppc64le))
	assert.Contains(t, AffinityNodeRequirementForAllArches(), linuxRequirement())
}

func TestTolerationsForAllArches(t *testing.T) {
	assert.Equal(t, TolerationForAllArches(), tolerationsForArches(amd64, arm64, ppc64le))
	assert.Contains(t, TolerationForAllArches(), armToleration())
	assert.Contains(t, TolerationForAllArches(), amdToleration())
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
