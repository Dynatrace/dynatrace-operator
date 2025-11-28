package attributes

import podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"

const (
	deprecatedClusterIDKey = "dt.kubernetes.cluster.id"
)

func setDeprecatedAttributes(attrs podattr.Attributes) podattr.Attributes {
	if attrs.UserDefined == nil {
		attrs.UserDefined = map[string]string{}
	}

	attrs.UserDefined[deprecatedClusterIDKey] = attrs.ClusterUID

	return attrs
}
