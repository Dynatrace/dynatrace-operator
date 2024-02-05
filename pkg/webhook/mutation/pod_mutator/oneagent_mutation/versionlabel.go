package oneagent_mutation

import (
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
)

const (
	versionMappingAnnotationName = "mapping.release.dynatrace.com/version"
	productMappingAnnotationName = "mapping.release.dynatrace.com/product"
	stageMappingAnnotationName   = "mapping.release.dynatrace.com/stage"
	buildVersionAnnotationName   = "mapping.release.dynatrace.com/build-version"
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
	annotationLabelMap := map[string]string{
		versionMappingAnnotationName: releaseVersionEnv,
		productMappingAnnotationName: releaseProductEnv,
		stageMappingAnnotationName:   releaseStageEnv,
		buildVersionAnnotationName:   releaseBuildVersionEnv,
	}

	versionLabelMapping := VersionLabelMapping{}

	for annotationKey, labelKey := range annotationLabelMap {
		if fieldRef, ok := namespace.Annotations[annotationKey]; ok {
			versionLabelMapping[labelKey] = fieldRef
		}
	}

	return versionLabelMapping
}

func mergeMappingWithDefault(labelMapping VersionLabelMapping) VersionLabelMapping {
	result := VersionLabelMapping{}
	maps.Copy(result, defaultVersionLabelMapping)
	maps.Copy(result, labelMapping)

	return result
}
