package metadata

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

func mutateUserContainers(request *dtwebhook.BaseRequest) {
	newContainers := request.NewContainers(ContainerIsInjected)
	for i := range newContainers {
		container := newContainers[i]
		setupVolumeMountsForUserContainer(container)
	}
}

func reinvokeUserContainers(request *dtwebhook.BaseRequest) bool {
	var updated bool

	newContainers := request.NewContainers(ContainerIsInjected)

	if len(newContainers) == 0 {
		return false
	}

	for i := range newContainers {
		container := newContainers[i]
		setupVolumeMountsForUserContainer(container)

		updated = true
	}

	return updated
}

func updateInstallContainer(installContainer *corev1.Container, workload *workloadInfo, entityID, clusterName string) {
	addInjectedEnv(installContainer)
	addDTClusterEnvs(installContainer, entityID, clusterName)
	addWorkloadInfoEnvs(installContainer, workload)
	addWorkloadEnrichmentInstallVolumeMount(installContainer)
}
