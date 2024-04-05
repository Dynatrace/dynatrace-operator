package metadata

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	dtwebhookutil "github.com/Dynatrace/dynatrace-operator/pkg/webhook/util"
	corev1 "k8s.io/api/core/v1"
)

func mutateUserContainers(request *dtwebhook.BaseRequest) {
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]

		if dtwebhookutil.IsContainerExcludedFromInjection(request, container.Name) {
			log.Info("Container excluded from metadata-enrichment injection", "container", container.Name)

			continue
		}

		setupVolumeMountsForUserContainer(container)
	}
}

func reinvokeUserContainers(request *dtwebhook.BaseRequest) bool {
	var updated bool

	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]
		if dtwebhookutil.IsContainerExcludedFromInjection(request, container.Name) {
			log.Info("Container excluded from metadata enrichment injection", "container", container.Name)

			continue
		}

		if containerIsInjected(container) {
			log.Info("Container already injected for metadata enrichment", "container", container.Name)

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
