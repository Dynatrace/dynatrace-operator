package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinity(t *testing.T) {
	dsInfo := builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "14",
	}
	affinity := dsInfo.affinity()
	assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})
	assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})

	dsInfo = builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "15",
	}
	affinity = dsInfo.affinity()
	assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})
	assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})

	dsInfo = builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "13",
	}
	affinity = dsInfo.affinity()
	assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})
	assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
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
	})
}
