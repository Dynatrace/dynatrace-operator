package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
)

func (dsInfo *builderInfo) affinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: dsInfo.affinityNodeSelectorTerms(),
			},
		},
	}
}

func (dsInfo *builderInfo) affinityNodeSelectorTerms() []corev1.NodeSelectorTerm {
	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		kubernetesArchOsSelectorTerm(),
	}

	return nodeSelectorTerms
}

func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: kubeobjects.AffinityNodeRequirementWithARM64(),
	}
}
