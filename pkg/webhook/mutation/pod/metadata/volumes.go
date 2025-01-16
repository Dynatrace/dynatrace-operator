package metadata

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

func setupVolumes(pod *corev1.Pod) {
	addIngestEndpointVolume(pod)
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

func setupVolumeMountsForUserContainer(container *corev1.Container) {
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
