package metadata

import podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"

const (
	deprecatedWorkloadKindKey = "dt.kubernetes.workload.kind"
	deprecatedWorkloadNameKey = "dt.kubernetes.workload.name"
)

func setDeprecatedAttributes(attrs *podattr.Attributes) {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[deprecatedWorkloadKindKey] = attrs.WorkloadKind
	attrs.UserDefined[deprecatedWorkloadNameKey] = attrs.WorkloadName
}
