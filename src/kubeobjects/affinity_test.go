package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	nodeSelectorRequirements = 2

	testArch1 = "arch1"
	testArch2 = "arch2"
)

func TestAffinityNodeRequirement(t *testing.T) {
	assert.Equal(t, AffinityNodeRequirement(), affinityNodeRequirementsForArches(amd64))
	assert.Equal(t, AffinityNodeRequirementWithARM64(), affinityNodeRequirementsForArches(amd64, arm64))
	assert.Equal(t, len(AffinityNodeRequirement()), nodeSelectorRequirements)

	assert.Contains(t, AffinityNodeRequirement(), linuxRequirement())
	assert.Contains(t, AffinityNodeRequirementWithARM64(), linuxRequirement())
}

func TestLinuxRequirement(t *testing.T) {
	expectedRequirement := corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{linux},
	}

	assert.Equal(t, expectedRequirement, linuxRequirement())
}

func TestArchRequirement(t *testing.T) {
	expectedRequirement := corev1.NodeSelectorRequirement{
		Key:      kubernetesArch,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{testArch1},
	}

	assert.Equal(t, expectedRequirement, archRequirement(testArch1))

	expectedRequirement = corev1.NodeSelectorRequirement{
		Key:      kubernetesArch,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{testArch1, testArch2},
	}

	assert.Equal(t, expectedRequirement, archRequirement(testArch1, testArch2))
}

func TestAffinityNodeRequirementWithoutArch(t *testing.T) {
	expectedAffinity := []corev1.NodeSelectorRequirement{
		linuxRequirement(),
	}
	affinity := AffinityNodeRequirementWithoutArch()

	assert.Equal(t, expectedAffinity, affinity)
}
