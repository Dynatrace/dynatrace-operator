// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package attributes

const (
	DeprecatedWorkloadKindKey = "dt.kubernetes.workload.kind"
	DeprecatedWorkloadNameKey = "dt.kubernetes.workload.name"
	DeprecatedClusterIDKey    = "dt.kubernetes.cluster.id"
)

func (attrs *Pod) applyDeprecatedAttributes() {
	attrs.deprecated[DeprecatedWorkloadKindKey] = attrs.workloadInfo[K8sWorkloadKindAttr]
	attrs.deprecated[DeprecatedWorkloadNameKey] = attrs.workloadInfo[K8sWorkloadNameAttr]
	attrs.deprecated[DeprecatedClusterIDKey] = attrs.clusterInfo[K8sClusterUIDAttr]
}
