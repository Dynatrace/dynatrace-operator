package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	corev1 "k8s.io/api/core/v1"
)

func (b *builder) affinity() *corev1.Affinity {
	var affinity corev1.Affinity
	if b.dk.Status.OneAgent.VersionStatus.Source == status.TenantRegistryVersionSource || b.dk.Status.OneAgent.VersionStatus.Source == status.CustomVersionVersionSource {
		affinity = node.AMDOnlyAffinity()
	} else {
		affinity = node.Affinity()
	}

	if b.hostInjectSpec != nil {
		if b.hostInjectSpec.NodeAffinity != nil {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
				affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
				b.hostInjectSpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms...,
			)

			affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
				affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
				b.hostInjectSpec.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution...,
			)
		}
	}

	return &affinity
}
