package dataingest_mutation

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	dtwebhookutil "github.com/Dynatrace/dynatrace-operator/pkg/webhook/util"
	corev1 "k8s.io/api/core/v1"
)

func mutateUserContainers(request *dtwebhook.BaseRequest) {
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]

		if !dtwebhookutil.ContainerIsExcluded(request, container.Name) {
			setupVolumeMountsForUserContainer(container)
		}
	}
}

func reinvokeUserContainers(request *dtwebhook.BaseRequest) bool {
	var updated bool
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]
		if dtwebhookutil.ContainerIsExcluded(request, container.Name) {
			continue
		}
		if containerIsInjected(container) {
			continue
		}
		setupVolumeMountsForUserContainer(container)
		updated = true
	}
	return updated
}

func updateInstallContainer(installContainer *corev1.Container, workload *workloadInfo) {
	addWorkloadInfoEnvs(installContainer, workload)
	addWorkloadEnrichmentVolumeMount(installContainer)
}
