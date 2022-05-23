package dataingest_mutation

import corev1 "k8s.io/api/core/v1"

func (mutator *DataIngestPodMutator) updateContainers(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		addEnrichmentVolumeMount(container)
	}
}

func updateInstallContainer(initContainer *corev1.Container, workload *workloadInfo) {
	addWorkloadInfoEnvs(initContainer, workload)
	addEnrichmentVolumeMount(initContainer)
}
