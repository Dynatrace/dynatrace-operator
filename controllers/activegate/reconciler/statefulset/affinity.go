package statefulset

// import (
// 	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
// 	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
// 	corev1 "k8s.io/api/core/v1"
// )

// const (
// 	kubernetesWithBetaVersion = 1.14
// )

// func affinity(stsProperties *statefulSetProperties) *corev1.Affinity {
// 	return &corev1.Affinity{
// 		NodeAffinity: &corev1.NodeAffinity{
// 			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
// 				NodeSelectorTerms: affinityNodeSelectorTerms(stsProperties),
// 			},
// 		},
// 	}
// }

// func affinityNodeSelectorTerms(stsProperties *statefulSetProperties) []corev1.NodeSelectorTerm {
// 	nodeSelectorTerms := []corev1.NodeSelectorTerm{
// 		kubernetesArchOsSelectorTerm(),
// 	}

// 	if kubesystem.KubernetesVersionAsFloat(stsProperties.majorKubernetesVersion, stsProperties.minorKubernetesVersion) < kubernetesWithBetaVersion {
// 		nodeSelectorTerms = append(nodeSelectorTerms, kubernetesBetaArchOsSelectorTerm())
// 	}

// 	return nodeSelectorTerms
// }

// func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
// 	return corev1.NodeSelectorTerm{
// 		MatchExpressions: kubeobjects.AffinityNodeRequirement(),
// 	}
// }

// func kubernetesBetaArchOsSelectorTerm() corev1.NodeSelectorTerm {
// 	return corev1.NodeSelectorTerm{
// 		MatchExpressions: kubeobjects.AffinityBetaNodeRequirement(),
// 	}
// }
