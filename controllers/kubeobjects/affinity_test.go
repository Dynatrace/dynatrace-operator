package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const nodeSelectorRequirements = 2

func TestAffinityNodeRequirement(t *testing.T) {
	assert.Equal(t, AffinityNodeRequirement(), affinityNodeRequirementsForArches(amd64))
	assert.Equal(t, AffinityNodeRequirementWithARM64(), affinityNodeRequirementsForArches(amd64, arm64))
	assert.Equal(t, len(AffinityNodeRequirement()), nodeSelectorRequirements)

	assert.Contains(t, AffinityNodeRequirement(), linuxRequirement())
	assert.Contains(t, AffinityNodeRequirementWithARM64(), linuxRequirement())
}

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{linux},
	}
}
