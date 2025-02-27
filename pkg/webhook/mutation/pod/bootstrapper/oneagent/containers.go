package oneagent

import (
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/sharedoneagent"
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

	installPath := maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	dk := request.DynaKube

	addVolumeMounts(container, installPath)
	sharedoneagent.AddDeploymentMetadataEnv(container, dk, mut.clusterID)
	sharedoneagent.AddPreloadEnv(container, installPath)

	if dk.FeatureLabelVersionDetection() {
		sharedoneagent.AddVersionDetectionEnvs(container, sharedoneagent.NewVersionLabelMapping(request.Namespace))
	}
}
