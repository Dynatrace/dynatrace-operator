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

	var nodeAffinitySpec corev1.NodeAffinity
	if b.dk.OneAgent.classicFullStack.nodeAffinity {
		nodeAffinitySpec = b.dk.OneAgent.classicFullStack.nodeAffinity
	} else if b.dk.OneAgent.hostMonitoring.nodeAffinity {
		nodeAffinitySpec = b.dk.OneAgent.hostMonitoring.nodeAffinity
	}

	if nodeAffinitySpec {
		if nodeAffinitySpec.RequiredDuringSchedulingIgnoredDuringExecution {
			append(
				affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
				nodeAffinitySpec.RequiredDuringSchedulingIgnoredDuringExecution...,
			)
		}
		if nodeAffinitySpec.PreferredDuringSchedulingIgnoredDuringExecution {
			append(
				affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
				nodeAffinitySpec.PreferredDuringSchedulingIgnoredDuringExecution...,
			)
		}
	}

	return &affinity
}
