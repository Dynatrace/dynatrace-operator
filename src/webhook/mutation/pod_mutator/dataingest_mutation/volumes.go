package dataingest_mutation

import (
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
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
					SecretName: dtingestendpoint.SecretEndpointName,
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
	addWorkloadEnrichmentVolumeMount(container)
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

func addWorkloadEnrichmentVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      workloadEnrichmentVolumeName,
			MountPath: standalone.EnrichmentPath,
		},
	)
}
