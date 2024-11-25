package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	corev1 "k8s.io/api/core/v1"
)

func (b *builder) affinity() *corev1.Affinity {
	var affinity corev1.Affinity
	if b.dk.Status.OneAgent.VersionStatus.Source == status.TenantRegistryVersionSource {
		affinity = node.AMDOnlyAffinity()
	} else {
		affinity = node.Affinity()
	}
	return &affinity
}
