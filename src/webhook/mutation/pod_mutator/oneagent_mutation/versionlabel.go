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
	return mergeMappingWithDefault(getMappingFromNamespace(namespace))
}

func getMappingFromNamespace(namespace corev1.Namespace) VersionLabelMapping {
	// TODO: Implementation => Parse Namespace annotations into correct format
	return VersionLabelMapping{}
}

func mergeMappingWithDefault(labelMapping VersionLabelMapping) VersionLabelMapping {
	// TODO: Implementation => Combine the maps
	return defaultVersionLabelMapping
}
