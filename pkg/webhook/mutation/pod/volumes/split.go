package volumes

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

const (
	oneagentDirPath              = "/oneagent"
	enrichmentJSONFilePath       = "/enrichment/dt_metadata.json"
	enrichmentPropertiesFilePath = "/enrichment/dt_metadata.properties"
	enrichmentEndpointDirPath    = "/enrichment/endpoint"
)

var (
	configOneAgentMountPath             = filepath.Join(ConfigMountPath, oneagentDirPath)
	configEnrichmentJSONMountPath       = filepath.Join(ConfigMountPath, enrichmentJSONFilePath)
	configEnrichmentPropertiesMountPath = filepath.Join(ConfigMountPath, enrichmentPropertiesFilePath)
	configEnrichmentEndpointMountPath   = filepath.Join(ConfigMountPath, enrichmentEndpointDirPath)
)

func HasSplitEnrichmentMounts(container *corev1.Container) bool {
	return k8smount.ContainsPath(container.VolumeMounts, configEnrichmentJSONMountPath) &&
		k8smount.ContainsPath(container.VolumeMounts, configEnrichmentPropertiesMountPath) &&
		k8smount.ContainsPath(container.VolumeMounts, configEnrichmentEndpointMountPath)
}

func HasSplitOneAgentMounts(container *corev1.Container) bool {
	return k8smount.ContainsPath(container.VolumeMounts, configOneAgentMountPath)
}

func addSplitMounts(container *corev1.Container, request *dtwebhook.BaseRequest) {
	if request.DynaKube.OneAgent().IsAppInjectionNeeded() {
		addSplitOneAgentConfigVolumeMount(container)
	}

	if request.DynaKube.MetadataEnrichment().IsEnabled() {
		addSplitEnrichmentConfigVolumeMount(container)
	}
}

func addSplitOneAgentConfigVolumeMount(container *corev1.Container) {
	vm := corev1.VolumeMount{
		Name:      ConfigVolumeName,
		MountPath: configOneAgentMountPath,
		SubPath:   configOneAgentSubPath(container.Name),
	}
	container.VolumeMounts = k8smount.Append(container.VolumeMounts, vm)
}

func addSplitEnrichmentConfigVolumeMount(container *corev1.Container) {
	vms := []corev1.VolumeMount{
		{
			Name:      ConfigVolumeName,
			MountPath: configEnrichmentJSONMountPath,
			SubPath:   configEnrichmentJSONSubPath(container.Name),
		},
		{
			Name:      ConfigVolumeName,
			MountPath: configEnrichmentPropertiesMountPath,
			SubPath:   configEnrichmentPropertiesSubPath(container.Name),
		},
		{
			Name:      ConfigVolumeName,
			MountPath: configEnrichmentEndpointMountPath,
			SubPath:   configEnrichmentEndpointsSubPath(container.Name),
		},
	}
	container.VolumeMounts = k8smount.Append(container.VolumeMounts, vms...)
}

func configOneAgentSubPath(containerName string) string {
	return filepath.Join(containerName, oneagentDirPath)
}

func configEnrichmentJSONSubPath(containerName string) string {
	return filepath.Join(containerName, enrichmentJSONFilePath)
}

func configEnrichmentPropertiesSubPath(containerName string) string {
	return filepath.Join(containerName, enrichmentPropertiesFilePath)
}

func configEnrichmentEndpointsSubPath(containerName string) string {
	return filepath.Join(containerName, enrichmentEndpointDirPath)
}
