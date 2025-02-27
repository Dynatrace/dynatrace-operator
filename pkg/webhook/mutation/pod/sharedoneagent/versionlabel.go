package sharedoneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
)

const (
	releaseVersionEnv      = "DT_RELEASE_VERSION"
	releaseProductEnv      = "DT_RELEASE_PRODUCT"
	releaseStageEnv        = "DT_RELEASE_STAGE"
	releaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

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

func NewVersionLabelMapping(namespace corev1.Namespace) VersionLabelMapping {
	return mergeMappingWithDefault(getMappingFromNamespace(namespace))
}

func AddVersionDetectionEnvs(container *corev1.Container, labelMapping VersionLabelMapping) {
	for envName, fieldPath := range labelMapping {
		if env.IsIn(container.Env, envName) {
			continue
		}

		container.Env = append(container.Env,
			corev1.EnvVar{
				Name: envName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fieldPath,
					},
				},
			},
		)
	}
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
