package daemonset

import corev1 "k8s.io/api/core/v1"

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

	if dsInfo.kubernetesVersion() < 1.14 {
		nodeSelectorTerms = append(nodeSelectorTerms, kubernetesBetaArchOsSelectorTerm())
	}

	return nodeSelectorTerms
}

func kubernetesBetaArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "beta.kubernetes.io/arch",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"amd64", "arm64"},
			},
			{
				Key:      "beta.kubernetes.io/os",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"linux"},
			},
		},
	}
}

func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "kubernetes.io/arch",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"amd64", "arm64"},
			},
			{
				Key:      "kubernetes.io/os",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"linux"},
			},
		},
	}
}
