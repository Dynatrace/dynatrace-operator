package node

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	corev1 "k8s.io/api/core/v1"
)

const (
	kubernetesArch = "kubernetes.io/arch"
	kubernetesOS   = "kubernetes.io/os"
)

func Affinity() corev1.Affinity {
	return AffinityForArches(arch.AMDImage, arch.ARMImage, arch.PPCLEImage, arch.S390Image)
}

// AMDOnlyAffinity provides an affinity that will only allow deployment on AMD64 nodes.
// This if manly needed for the Dynatrace tenant-registry as it only has AMD64 images.
func AMDOnlyAffinity() corev1.Affinity {
	return AffinityForArches(arch.AMDImage)
}

func AffinityForArches(arches ...string) corev1.Affinity {
	return corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: affinityNodeRequirementsForArches(arches...),
					},
				},
			},
		},
	}
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
