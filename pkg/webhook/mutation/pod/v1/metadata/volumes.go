package metadata

import (
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

func setupVolumes(pod *corev1.Pod) {
	addIngestEndpointVolume(pod)
	addWorkloadEnrichmentVolume(pod)
}

func addIngestEndpointVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: ingestEndpointVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: consts.EnrichmentEndpointSecretName,
				},
			},
		},
	)
}

func addWorkloadEnrichmentVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: workloadEnrichmentVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func setupVolumeMountsForUserContainer(container *corev1.Container) {
	addWorkloadEnrichmentVolumeMount(container, consts.EnrichmentPropertiesFilename, fmt.Sprintf(consts.EnrichmentInitPropertiesFilenameTemplate, container.Name))
	addWorkloadEnrichmentVolumeMount(container, consts.EnrichmentJSONFilename, fmt.Sprintf(consts.EnrichmentInitJSONFilenameTemplate, container.Name))
	addEnrichmentEndpointVolumeMount(container)
}

func addEnrichmentEndpointVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ingestEndpointVolumeName,
			MountPath: enrichmentEndpointPath,
		},
	)
}

func addWorkloadEnrichmentVolumeMount(container *corev1.Container, destFilename, initFilename string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      workloadEnrichmentVolumeName,
			MountPath: filepath.Join(consts.EnrichmentMountPath, destFilename),
			SubPath:   initFilename,
		},
	)
}

func addWorkloadEnrichmentInstallVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      workloadEnrichmentVolumeName,
			MountPath: consts.EnrichmentInitPath,
		},
	)
}
