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

func (mutator *OneAgentPodMutator) updateContainers(request *dtwebhook.MutationRequest) {
	for i := range request.Pod.Spec.Containers {
		container := &request.Pod.Spec.Containers[i]
		addContainerInfoInitEnv(request.InitContainer, i+1, container.Name, container.Image)
		mutator.addOneAgentToContainer(request.Pod, request.DynaKube, container)
	}
}

func (mutator *OneAgentPodMutator) addOneAgentToContainer(pod *corev1.Pod, dynakube *dynatracev1beta1.DynaKube, container *corev1.Container) {

	log.Info("updating container with missing preload variables", "containerName", container.Name)
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
