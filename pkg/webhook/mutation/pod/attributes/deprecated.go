package attributes

const (
	DeprecatedWorkloadKindKey = "dt.kubernetes.workload.kind"
	DeprecatedWorkloadNameKey = "dt.kubernetes.workload.name"
	DeprecatedClusterIDKey    = "dt.kubernetes.cluster.id"
)

func (attrs *PodAttributes) GetDeprecatedAttributes() {
	attrs.deprecated[DeprecatedWorkloadKindKey] = attrs.workloadInfo[K8sWorkloadKindAttr]
	attrs.deprecated[DeprecatedWorkloadNameKey] = attrs.workloadInfo[K8sWorkloadNameAttr]
	attrs.deprecated[DeprecatedClusterIDKey] = attrs.clusterInfo[K8sClusterUIDAttr]
}
