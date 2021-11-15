package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	corev1 "k8s.io/api/core/v1"
)

const (
	kubernetesWithBetaVersion = 1.14
)

func affinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: affinityNodeSelectorTerms(),
			},
		},
	}
}

func affinityNodeSelectorTerms() []corev1.NodeSelectorTerm {
	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		kubernetesArchOsSelectorTerm(),
	}
	return nodeSelectorTerms
}

func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: kubeobjects.AffinityNodeRequirement(),
	}
}
