package volumes

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

const (
	oneagentDirPath              = "/oneagent"
	enrichmentJSONFilePath       = "/enrichment/dt_metadata.json"
	enrichmentPropertiesFilePath = "/enrichment/dt_metadata.properties"
	enrichmentEndpointsDirPath   = "/enrichment/endpoints"
)

var (
	configOneAgentMountPath             = filepath.Join(ConfigMountPath, oneagentDirPath)
	configEnrichmentJSONMountPath       = filepath.Join(ConfigMountPath, enrichmentJSONFilePath)
	configEnrichmentPropertiesMountPath = filepath.Join(ConfigMountPath, enrichmentPropertiesFilePath)
	configEnrichmentEndpointsMountPath  = filepath.Join(ConfigMountPath, enrichmentEndpointsDirPath)
)

func HasSplitEnrichmentMounts(container *corev1.Container) bool {
	return mounts.IsPathIn(container.VolumeMounts, configEnrichmentJSONMountPath) ||
		mounts.IsPathIn(container.VolumeMounts, configEnrichmentPropertiesMountPath) ||
		mounts.IsPathIn(container.VolumeMounts, configEnrichmentEndpointsMountPath)
}

func HasSplitOneAgentMounts(container *corev1.Container) bool {
	return mounts.IsPathIn(container.VolumeMounts, configOneAgentMountPath)
}

func addSplitMounts(container *corev1.Container, request *dtwebhook.BaseRequest) {
	if request.DynaKube.OneAgent().IsAppInjectionNeeded() {
		addSplitOneAgentConfigVolumeMount(container)
	}

	if request.DynaKube.MetadataEnrichmentEnabled() {
		addSplitEnrichmentConfigVolumeMount(container)
	}
}

func addSplitOneAgentConfigVolumeMount(container *corev1.Container) {
	vm := corev1.VolumeMount{
		Name:      ConfigVolumeName,
		MountPath: configOneAgentMountPath,
		SubPath:   configOneAgentSubPath(container.Name),
	}
	container.VolumeMounts = mounts.Append(container.VolumeMounts, vm)
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
			MountPath: configEnrichmentEndpointsMountPath,
			SubPath:   configEnrichmentEndpointsSubPath(container.Name),
		},
	}
	container.VolumeMounts = mounts.Append(container.VolumeMounts, vms...)
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
	return filepath.Join(containerName, enrichmentEndpointsDirPath)
}
