package kubeobjects

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	kubernetesArch = "kubernetes.io/arch"
	kubernetesOS   = "kubernetes.io/os"

	amd64   = "amd64"
	arm64   = "arm64"
	ppc64le = "ppc64le"
	linux   = "linux"
)

func TolerationForSupportedArches() []corev1.Toleration {
	return tolerationsForArches(amd64, arm64, ppc64le)
}

func AffinityNodeRequirementForSupportedArches() []corev1.NodeSelectorRequirement {
	return affinityNodeRequirementsForArches(amd64, arm64, ppc64le)
}

func tolerationsForArches(arches ...string) []corev1.Toleration {
	tolerations := make([]corev1.Toleration, 0)
	for _, arch := range arches {
		tolerations = append(tolerations, corev1.Toleration{
			Key:      kubernetesArch,
			Operator: corev1.TolerationOpEqual,
			Value:    arch,
			Effect:   corev1.TaintEffectNoSchedule,
		})
	}
	return tolerations
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
			Values:   []string{linux},
		},
	}
}
