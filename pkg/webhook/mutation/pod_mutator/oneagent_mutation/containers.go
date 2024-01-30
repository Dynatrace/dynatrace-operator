package oneagent_mutation

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	dtwebhookutil "github.com/Dynatrace/dynatrace-operator/pkg/webhook/util"
	corev1 "k8s.io/api/core/v1"
)

func (mutator *OneAgentPodMutator) configureInitContainer(request *dtwebhook.MutationRequest, installer installerInfo) {
	addInstallerInitEnvs(request.InstallContainer, installer)
	addInitVolumeMounts(request.InstallContainer, request.DynaKube)
}

func (mutator *OneAgentPodMutator) setContainerCount(initContainer *corev1.Container, containerCount int) {
	desiredContainerCountEnvVarValue := strconv.Itoa(containerCount)
	initContainer.Env = env.AddOrUpdate(initContainer.Env, corev1.EnvVar{Name: consts.AgentContainerCountEnv, Value: desiredContainerCountEnvVarValue})
}

func (mutator *OneAgentPodMutator) mutateUserContainers(request *dtwebhook.MutationRequest) int {
	injectedContainers := 0

	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]

		if dtwebhookutil.IsContainerExcludedFromInjection(request.BaseRequest, container.Name) {
			log.Info("Container excluded from code modules ingest injection", "container", container.Name)

			continue
		}

		addContainerInfoInitEnv(request.InstallContainer, i+1, container.Name, container.Image)
		mutator.addOneAgentToContainer(request.ToReinvocationRequest(), container)

		injectedContainers++
	}

	return injectedContainers
}

// reinvokeUserContainers mutates each user container that hasn't been injected yet.
// It makes sure that the new containers will have an environment variable in the install-container
// that doesn't conflict with the previous environment variables of the originally injected containers
func (mutator *OneAgentPodMutator) reinvokeUserContainers(request *dtwebhook.ReinvocationRequest) bool {
	pod := request.Pod
	oneAgentInstallContainer := findOneAgentInstallContainer(pod.Spec.InitContainers)
	newContainers := []*corev1.Container{}

	injectedContainers := 0

	for i := range pod.Spec.Containers {
		currentContainer := &pod.Spec.Containers[i]
		if dtwebhookutil.IsContainerExcludedFromInjection(request.BaseRequest, currentContainer.Name) {
			log.Info("Container excluded from code modules ingest injection", "container", currentContainer.Name)

			continue
		}

		if containerIsInjected(currentContainer) {
			injectedContainers++

			continue
		}

		newContainers = append(newContainers, currentContainer)
	}

	if len(newContainers) == 0 {
		return false
	}

	for i := range newContainers {
		currentContainer := newContainers[i]
		addContainerInfoInitEnv(oneAgentInstallContainer, injectedContainers+i+1, currentContainer.Name, currentContainer.Image)
		mutator.addOneAgentToContainer(request, currentContainer)
	}

	mutator.setContainerCount(oneAgentInstallContainer, injectedContainers+len(newContainers))

	return true
}

func (mutator *OneAgentPodMutator) addOneAgentToContainer(request *dtwebhook.ReinvocationRequest, container *corev1.Container) {
	log.Info("adding OneAgent to container", "name", container.Name)

	installPath := maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	dynakube := request.DynaKube

	addOneAgentVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dynakube, mutator.clusterID)
	addPreloadEnv(container, installPath)

	addCertVolumeMounts(container, dynakube)

	if dynakube.FeatureAgentInitialConnectRetry() > 0 {
		addCurlOptionsVolumeMount(container)
	}

	if dynakube.Spec.NetworkZone != "" {
		addNetworkZoneEnv(container, dynakube.Spec.NetworkZone)
	}

	if dynakube.FeatureLabelVersionDetection() {
		addVersionDetectionEnvs(container, newVersionLabelMapping(request.Namespace))
	}

	if dynakube.FeatureReadOnlyCsiVolume() {
		addVolumeMountsForReadOnlyCSI(container)
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
