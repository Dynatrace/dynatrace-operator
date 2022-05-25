package oneagent_mutation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

func (mutator *OneAgentPodMutator) configureInitContainer(request *dtwebhook.MutationRequest, installer installerInfo) {
	addInstallerInitEnvs(request.InitContainer, installer, mutator.getVolumeMode(request.DynaKube))
	addInitVolumeMounts(request.InitContainer)
}

func (mutator *OneAgentPodMutator) mutateUserContainers(request *dtwebhook.MutationRequest) {
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]
		addContainerInfoInitEnv(request.InitContainer, i+1, container.Name, container.Image)
		mutator.addOneAgentToContainer(request.Pod, request.DynaKube, container)
	}
}

// reinvokeUserContainers mutates each user container that hasn't been injected
// it does it in an way to make sure that the new containers will have an envvar in the install-container
// that don't conflict with the previous envvars for the originally injected containers
func (mutator *OneAgentPodMutator) reinvokeUserContainers(request *dtwebhook.ReinvocationRequest) {
	pod := request.Pod
	initContainer := dtwebhook.FindInitContainer(pod.Spec.InitContainers)
	newContainers := []*corev1.Container{}

	for i := range pod.Spec.Containers {
		currentContainer := &pod.Spec.Containers[i]
		if containerIsInjected(currentContainer) {
			continue
		}
		newContainers = append(newContainers, currentContainer)
	}

	oldContainersLen := len(pod.Spec.Containers) - len(newContainers)
	for i := range newContainers {
		currentContainer := newContainers[i]
		addContainerInfoInitEnv(initContainer, oldContainersLen+i+1, currentContainer.Name, currentContainer.Image)
		mutator.addOneAgentToContainer(request.Pod, request.DynaKube, currentContainer)
	}
}

func (mutator *OneAgentPodMutator) addOneAgentToContainer(pod *corev1.Pod, dynakube *dynatracev1beta1.DynaKube, container *corev1.Container) {

	log.Info("adding OneAgent to container", "container", container.Name)
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	addOneAgentVolumeMounts(container, installPath)
	if dynakube.HasActiveGateCaCert() {
		addCertVolumeMounts(container)
	}

	addInitialConnectRetryEnv(container, dynakube)

	addDeploymentMetadataEnv(container, dynakube, mutator.clusterID)
	addPreloadEnv(container, installPath)

	if dynakube.NeedsOneAgentProxy() {
		addProxyEnv(container)
	}

	if dynakube.Spec.NetworkZone != "" {
		addNetworkZoneEnv(container, dynakube.Spec.NetworkZone)
	}
}
