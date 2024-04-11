package node

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	corev1 "k8s.io/api/core/v1"
)

const (
	kubernetesArch = "kubernetes.io/arch"
	kubernetesOS   = "kubernetes.io/os"
)

func AffinityNodeRequirementForSupportedArches() []corev1.NodeSelectorRequirement {
	return affinityNodeRequirementsForArches(arch.AMDImage, arch.ARMImage, arch.PPCLEImage, arch.S390Image)
}

func affinityNodeRequirementsForArches(arches ...string) []corev1.NodeSelectorRequirement {
	return []corev1.NodeSelectorRequirement{
		{
			Key:      kubernetesArch,
			Operator: corev1.NodeSelectorOpIn,
			Values:   arches,
		},
		{
			Key:      kubernetesOS,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{arch.DefaultImageOS},
		},
	}
}
