package attributes

import podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"

const (
	DeprecatedClusterIDKey    = "dt.kubernetes.cluster.id"
	DeprecatedWorkloadKindKey = "dt.kubernetes.workload.kind"
	DeprecatedWorkloadNameKey = "dt.kubernetes.workload.name"
)

func setDeprecatedClusterAttributes(attrs podattr.Attributes) podattr.Attributes {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[DeprecatedClusterIDKey] = attrs.ClusterUID

	return attrs
}

func setDeprecatedWorkloadAttributes(attrs podattr.Attributes) podattr.Attributes {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[DeprecatedWorkloadKindKey] = attrs.WorkloadKind
	attrs.UserDefined[DeprecatedWorkloadNameKey] = attrs.WorkloadName

	return attrs
}
