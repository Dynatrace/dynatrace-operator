package oneagent_mutation

import (
	corev1 "k8s.io/api/core/v1"
)

var (
	defaultVersionLabelMapping = VersionLabelMapping{
		releaseVersionEnv: "metadata.labels['app.kubernetes.io/version']",
		releaseProductEnv: "metadata.labels['app.kubernetes.io/part-of']",
	}
)

type VersionLabelMapping map[string]string

func newVersionLabelMapping(namespace corev1.Namespace) VersionLabelMapping {
	// TODO
	return mergeMappingWithDefault(getMappingFromNamespace(namespace))
}

func getMappingFromNamespace(namespace corev1.Namespace) VersionLabelMapping {
	// TODO
	return VersionLabelMapping{}
}

func mergeMappingWithDefault(labelMapping VersionLabelMapping) VersionLabelMapping {
	// TODO
	return defaultVersionLabelMapping
}
