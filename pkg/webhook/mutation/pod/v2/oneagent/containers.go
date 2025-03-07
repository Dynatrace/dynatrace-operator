package oneagent

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	corev1 "k8s.io/api/core/v1"
)

func (mut *Mutator) mutateUserContainers(request *dtwebhook.MutationRequest) {
	newContainers := request.NewContainers(ContainerIsInjected)

	for i := range newContainers {
		container := newContainers[i]
		mut.addOneAgentToContainer(request.ToReinvocationRequest(), container)
	}
}

func (mut *Mutator) reinvokeUserContainers(request *dtwebhook.ReinvocationRequest) bool {
	newContainers := request.NewContainers(ContainerIsInjected)

	if len(newContainers) == 0 {
		return false
	}

	for i := range newContainers {
		container := newContainers[i]
		mut.addOneAgentToContainer(request, container)
	}

	return true
}

func (mut *Mutator) addOneAgentToContainer(request *dtwebhook.ReinvocationRequest, container *corev1.Container) {
	log.Info("adding OneAgent to container", "name", container.Name)

	installPath := oacommon.DefaultInstallPath
	dk := request.DynaKube

	addVolumeMounts(container, installPath)
	oacommon.AddDeploymentMetadataEnv(container, dk)
	oacommon.AddPreloadEnv(container, installPath)
}
