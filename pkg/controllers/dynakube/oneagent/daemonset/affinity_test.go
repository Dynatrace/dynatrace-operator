package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinity(t *testing.T) {
	t.Run("none tenant-registry DynaKube has all the architectures", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Status.OneAgent.VersionStatus.Source = status.CustomImageVersionSource
		dsBuilder := builder{dk: &dk}
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
	})

	t.Run("tenant-registry DynaKube has only AMD architectures", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Status.OneAgent.VersionStatus.Source = status.TenantRegistryVersionSource
		dsBuilder := builder{dk: &dk}
		affinity := dsBuilder.affinity()
		assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64"},
				},
				{
					Key:      "kubernetes.io/os",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"linux"},
				},
			},
		})
	})
}
