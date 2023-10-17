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

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{linux},
	}
}
