package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	corev1 "k8s.io/api/core/v1"
)

func (b *builder) affinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: b.affinityNodeSelectorTerms(),
			},
		},
	}
}

func (b *builder) affinityNodeSelectorTerms() []corev1.NodeSelectorTerm {
	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		kubernetesArchOsSelectorTerm(),
	}

	return nodeSelectorTerms
}

func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: node.AffinityNodeRequirementForSupportedArches(),
	}
}
