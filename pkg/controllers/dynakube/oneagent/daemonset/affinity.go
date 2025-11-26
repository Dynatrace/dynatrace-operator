package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	corev1 "k8s.io/api/core/v1"
)

func (b *builder) affinity() *corev1.Affinity {
	var affinity corev1.Affinity
	if b.dk.Status.OneAgent.Source == status.TenantRegistryVersionSource || b.dk.Status.OneAgent.Source == status.CustomVersionVersionSource {
		affinity = k8saffinity.NewAMDOnlyNodeAffinity()
	} else {
		affinity = k8saffinity.NewMultiArchNodeAffinity()
	}

	return &affinity
}
