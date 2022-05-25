package dataingest_mutation

import corev1 "k8s.io/api/core/v1"

func mutateUserContainers(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		setupVolumeMountsForUserContainer(container)
	}
}

func reinvokeUserContainers(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if containerIsInjected(container) {
			continue
		}
		setupVolumeMountsForUserContainer(container)
	}
}

func updateInstallContainer(initContainer *corev1.Container, workload *workloadInfo) {
	addWorkloadInfoEnvs(initContainer, workload)
	addEnrichmentVolumeMount(initContainer)
}
