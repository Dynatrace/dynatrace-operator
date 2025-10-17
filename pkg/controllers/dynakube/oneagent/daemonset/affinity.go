package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/affinity"
	corev1 "k8s.io/api/core/v1"
)

func (b *builder) affinity() *corev1.Affinity {
	var af corev1.Affinity
	if b.dk.Status.OneAgent.Source == status.TenantRegistryVersionSource || b.dk.Status.OneAgent.Source == status.CustomVersionVersionSource {
		af = affinity.NewAMDOnlyNodeAffinity()
	} else {
		af = affinity.NewMultiArchNodeAffinity()
	}

	return &af
}
