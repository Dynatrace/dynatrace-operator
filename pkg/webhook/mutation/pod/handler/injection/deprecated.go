package injection

import podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"

const (
	deprecatedClusterIDKey = "dt.kubernetes.cluster.id"
)

func setDeprecatedAttributes(attrs *podattr.Attributes) {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[deprecatedClusterIDKey] = attrs.ClusterUID
}
