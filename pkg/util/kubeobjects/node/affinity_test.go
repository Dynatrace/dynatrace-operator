package node

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinityNodeRequirement(t *testing.T) {
	assert.Equal(t, AffinityNodeRequirementForSupportedArches(), affinityNodeRequirementsForArches(arch.AMDImageArch, arch.ARMImageArch, arch.PPCLEImageArch, arch.S390ImageArch))
	assert.Contains(t, AffinityNodeRequirementForSupportedArches(), linuxRequirement())
}

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{arch.DefaultImageOS},
	}
}
