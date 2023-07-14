package oneagent_mutation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OneAgentPodMutator struct {
	image            string
	clusterID        string
	webhookNamespace string
	client           client.Client
	apiReader        client.Reader
}

func NewOneAgentPodMutator(image, clusterID, webhookNamespace string, client client.Client, apiReader client.Reader) *OneAgentPodMutator { //nolint:revive // argument-limit doesn't apply to constructors
	return &OneAgentPodMutator{
		image:            image,
		clusterID:        clusterID,
		webhookNamespace: webhookNamespace,
		client:           client,
		apiReader:        apiReader,
	}
}

func (mutator *OneAgentPodMutator) Enabled(request *dtwebhook.BaseRequest) bool {
	return kubeobjects.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInject, request.DynaKube.FeatureAutomaticInjection())
}

func (mutator *OneAgentPodMutator) Injected(request *dtwebhook.BaseRequest) bool {
	return kubeobjects.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInjected, false)
}

func (mutator *OneAgentPodMutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Info("injecting OneAgent into pod", "podName", request.PodName())
	installerInfo := getInstallerInfo(request.Pod, request.DynaKube)
	mutator.addVolumes(request.Pod, request.DynaKube)
	mutator.configureInitContainer(request, installerInfo)
	mutator.setContainerCount(request.InstallContainer, len(request.Pod.Spec.Containers))
	mutator.mutateUserContainers(request)
	addInjectionConfigVolumeMount(request.InstallContainer)
	setInjectedAnnotation(request.Pod)
	return nil
}

func (mutator *OneAgentPodMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mutator.Injected(request.BaseRequest) {
		return false
	}
	log.Info("reinvoking", "podName", request.PodName())
	return mutator.reinvokeUserContainers(request)
}

func containerIsInjected(container *corev1.Container) bool {
	for _, e := range container.Env {
		if e.Name == dynatraceMetadataEnv {
			return true
		}
	}
	return false
}

func getVolumeMode(dynakube dynatracev1beta1.DynaKube) string {
	if dynakube.NeedsCSIDriver() {
		return string(config.AgentCsiMode)
	}
	return string(config.AgentInstallerMode)
}
