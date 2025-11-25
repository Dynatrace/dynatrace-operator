package k8saffinity

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	corev1 "k8s.io/api/core/v1"
)

const (
	kubernetesArch = "kubernetes.io/arch"
	kubernetesOS   = "kubernetes.io/os"
)

func NewMultiArchNodeAffinity() corev1.Affinity {
	return nodeAffinityForArches(arch.AMDImage, arch.ARMImage, arch.PPCLEImage, arch.S390Image)
}

// NewAMDOnlyNodeAffinity provides an affinity that will only allow deployment on AMD64 nodes.
// This is manly needed for the Dynatrace tenant-registry as it only has AMD64 images.
func NewAMDOnlyNodeAffinity() corev1.Affinity {
	return nodeAffinityForArches(arch.AMDImage)
}

func nodeAffinityForArches(arches ...string) corev1.Affinity {
	return corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: nodeAffinityRequirementsForArches(arches...),
					},
				},
			},
		},
	}
}

func nodeAffinityRequirementsForArches(arches ...string) []corev1.NodeSelectorRequirement {
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
