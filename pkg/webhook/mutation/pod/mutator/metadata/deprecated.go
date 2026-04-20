package metadata

import podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"

const (
	DeprecatedWorkloadKindKey = "dt.kubernetes.workload.kind"
	DeprecatedWorkloadNameKey = "dt.kubernetes.workload.name"
	DeprecatedClusterIDKey    = "dt.kubernetes.cluster.id"
)

func setDeprecatedAttributes(attrs *podattr.Attributes) {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[DeprecatedWorkloadKindKey] = attrs.WorkloadKind
	attrs.UserDefined[DeprecatedWorkloadNameKey] = attrs.WorkloadName
	attrs.UserDefined[DeprecatedClusterIDKey] = attrs.ClusterUID
}
