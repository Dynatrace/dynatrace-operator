package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinity(t *testing.T) {
	dsBuilder := builder{}
	affinity := dsBuilder.affinity()
	assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "beta.kubernetes.io/arch",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
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
				Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
			},
			{
				Key:      "kubernetes.io/os",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"linux"},
			},
		},
	})
}
