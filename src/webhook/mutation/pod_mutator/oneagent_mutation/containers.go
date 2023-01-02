package oneagent_mutation

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

func (mutator *OneAgentPodMutator) configureInitContainer(request *dtwebhook.MutationRequest, installer installerInfo) {
	addInstallerInitEnvs(request.InstallContainer, installer, mutator.getVolumeMode(request.DynaKube))
	addInitVolumeMounts(request.InstallContainer)
}

func (mutator *OneAgentPodMutator) setContainerCount(initContainer *corev1.Container, containerCount int) {
	desiredContainerCountEnvVarValue := strconv.Itoa(containerCount)
	initContainer.Env = kubeobjects.AddOrUpdate(initContainer.Env, corev1.EnvVar{Name: config.AgentContainerCountEnv, Value: desiredContainerCountEnvVarValue})
}

func (mutator *OneAgentPodMutator) mutateUserContainers(request *dtwebhook.MutationRequest) {
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]
		addContainerInfoInitEnv(request.InstallContainer, i+1, container.Name, container.Image)
		mutator.addOneAgentToContainer(request.ToReinvocationRequest(), container)
	}
}

// reinvokeUserContainers mutates each user container that hasn't been injected yet.
// It makes sure that the new containers will have an environment variable in the install-container
// that doesn't conflict with the previous environment variables of the originally injected containers
func (mutator *OneAgentPodMutator) reinvokeUserContainers(request *dtwebhook.ReinvocationRequest) bool {
	pod := request.Pod
	oneAgentInstallContainer := findOneAgentInstallContainer(pod.Spec.InitContainers)
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
		addContainerInfoInitEnv(oneAgentInstallContainer, oldContainersLen+i+1, currentContainer.Name, currentContainer.Image)
		mutator.addOneAgentToContainer(request, currentContainer)
	}

	if len(newContainers) == 0 {
		return false
	}

	mutator.setContainerCount(oneAgentInstallContainer, len(request.Pod.Spec.Containers))
	return true
}

func (mutator *OneAgentPodMutator) addOneAgentToContainer(request *dtwebhook.ReinvocationRequest, container *corev1.Container) {
	log.Info("adding OneAgent to container", "name", container.Name)
	installPath := kubeobjects.GetField(request.Pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	dynakube := request.DynaKube
	addOneAgentVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dynakube, mutator.clusterID)
	addPreloadEnv(container, installPath)

	if dynakube.HasActiveGateCaCert() {
		addCertVolumeMounts(container)
	}

	if dynakube.FeatureAgentInitialConnectRetry() > 0 {
		addCurlOptionsVolumeMount(container)
	}

	if dynakube.NeedsOneAgentProxy() {
		addProxyEnv(container)
	}

	if dynakube.Spec.NetworkZone != "" {
		addNetworkZoneEnv(container, dynakube.Spec.NetworkZone)
	}

	if dynakube.FeatureLabelVersionDetection() {
		addVersionDetectionEnvs(container, newVersionLabelMapping(request.Namespace))
	}
}

func findOneAgentInstallContainer(initContainers []corev1.Container) *corev1.Container {
	for i := range initContainers {
		container := &initContainers[i]
		if container.Name == dtwebhook.InstallContainerName {
			return container
		}
	}
	return nil
}
