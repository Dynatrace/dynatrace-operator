package oneagent

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	dtwebhookutil "github.com/Dynatrace/dynatrace-operator/pkg/webhook/util"
	corev1 "k8s.io/api/core/v1"
)

func (mut *Mutator) configureInitContainer(request *dtwebhook.MutationRequest, installer installerInfo) {
	addInstallerInitEnvs(request.InstallContainer, installer)
	addInitVolumeMounts(request.InstallContainer, request.DynaKube)
}

func (mut *Mutator) setContainerCount(initContainer *corev1.Container, containerCount int) {
	desiredContainerCountEnvVarValue := strconv.Itoa(containerCount)
	initContainer.Env = env.AddOrUpdate(initContainer.Env, corev1.EnvVar{Name: consts.AgentContainerCountEnv, Value: desiredContainerCountEnvVarValue})
}

func (mut *Mutator) mutateUserContainers(request *dtwebhook.MutationRequest) int {
	injectedContainers := 0

	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]

		if dtwebhookutil.IsContainerExcludedFromInjection(request.BaseRequest, container.Name) {
			log.Info("Container excluded from code modules ingest injection", "container", container.Name)

			continue
		}

		addContainerInfoInitEnv(request.InstallContainer, injectedContainers+1, container.Name, container.Image)
		mut.addOneAgentToContainer(request.ToReinvocationRequest(), container)

		injectedContainers++
	}

	return injectedContainers
}

// reinvokeUserContainers mutates each user container that hasn't been injected yet.
// It makes sure that the new containers will have an environment variable in the install-container
// that doesn't conflict with the previous environment variables of the originally injected containers
func (mut *Mutator) reinvokeUserContainers(request *dtwebhook.ReinvocationRequest) bool {
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
		mut.addOneAgentToContainer(request, currentContainer)
	}

	mut.setContainerCount(oneAgentInstallContainer, injectedContainers+len(newContainers))

	return true
}

func (mut *Mutator) addOneAgentToContainer(request *dtwebhook.ReinvocationRequest, container *corev1.Container) {
	log.Info("adding OneAgent to container", "name", container.Name)

	installPath := maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	dk := request.DynaKube

	addOneAgentVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dk, mut.clusterID)
	addPreloadEnv(container, installPath)

	addCertVolumeMounts(container, dk)

	if dk.FeatureAgentInitialConnectRetry() > 0 {
		addCurlOptionsVolumeMount(container)
	}

	if dk.Spec.NetworkZone != "" {
		addNetworkZoneEnv(container, dk.Spec.NetworkZone)
	}

	if dk.FeatureLabelVersionDetection() {
		addVersionDetectionEnvs(container, newVersionLabelMapping(request.Namespace))
	}

	if dk.FeatureReadOnlyCsiVolume() {
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
