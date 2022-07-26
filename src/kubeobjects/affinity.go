package kubeobjects

import corev1 "k8s.io/api/core/v1"

const (
	kubernetesArch = "kubernetes.io/arch"
	kubernetesOS   = "kubernetes.io/os"

	amd64 = "amd64"
	arm64 = "arm64"
	linux = "linux"
)

func AffinityNodeRequirement() []corev1.NodeSelectorRequirement {
	return affinityNodeRequirementsForArches(amd64)
}

func AffinityNodeRequirementWithARM64() []corev1.NodeSelectorRequirement {
	return affinityNodeRequirementsForArches(amd64, arm64)
}

func AffinityNodeRequirementWithoutArch() []corev1.NodeSelectorRequirement {
	return []corev1.NodeSelectorRequirement{linuxRequirement()}
}

func affinityNodeRequirementsForArches(arches ...string) []corev1.NodeSelectorRequirement {
	return []corev1.NodeSelectorRequirement{
		archRequirement(arches...),
		linuxRequirement(),
	}
}

func archRequirement(arches ...string) corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesArch,
		Operator: corev1.NodeSelectorOpIn,
		Values:   arches,
	}
}

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{linux},
	}
}
