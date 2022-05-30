package dataingest_mutation

import (
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	corev1 "k8s.io/api/core/v1"
)

func setupVolumes(pod *corev1.Pod) {
	addEnrichmentVolume(pod)
	addEnrichmentEndpointVolume(pod)
}

func addEnrichmentVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: EnrichmentVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtingestendpoint.SecretEndpointName,
				},
			},
		},
	)
}

func addEnrichmentEndpointVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: EnrichmentEndpointVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func setupVolumeMountsForUserContainer(container *corev1.Container) {
	addEnrichmentVolumeMount(container)
	addEnrichmentEndpointVolumeMount(container)
}

func addEnrichmentEndpointVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      EnrichmentEndpointVolumeName,
			MountPath: EnrichmentEndpointPath,
		},
	)
}

func addEnrichmentVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      EnrichmentVolumeName,
			MountPath: standalone.EnrichmentPath,
		},
	)
}
