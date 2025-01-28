package oneagent

import (
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

func (mut *Mutator) configureInitContainer(request *dtwebhook.MutationRequest, installer installerInfo) {
	addInstallerInitEnvs(request.InstallContainer, installer)
	addInitVolumeMounts(request.InstallContainer, request.DynaKube)
}

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

	installPath := maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	dk := request.DynaKube

	addOneAgentVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dk, mut.clusterID)
	addPreloadEnv(container, installPath)

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
