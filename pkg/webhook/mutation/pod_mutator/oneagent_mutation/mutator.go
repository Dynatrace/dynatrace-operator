package oneagent_mutation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	if !request.DynaKube.IsOneAgentCommunicationRouteClear() {
		log.Info("OneAgent were not yet able to communicate with tenant, no direct route or ready ActiveGate available, code modules have not been injected.")
		setNotInjectedAnnotations(request.Pod, dtwebhook.EmptyConnectionInfoReason)
		return nil
	}

	log.Info("injecting OneAgent into pod", "podName", request.PodName())
	if err := mutator.ensureInitSecret(request); err != nil {
		return err
	}

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

func (mutator *OneAgentPodMutator) ensureInitSecret(request *dtwebhook.MutationRequest) error {
	var initSecret corev1.Secret
	secretObjectKey := client.ObjectKey{Name: consts.AgentInitSecretName, Namespace: request.Namespace.Name}
	if err := mutator.apiReader.Get(request.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
		initGenerator := initgeneration.NewInitGenerator(mutator.client, mutator.apiReader, mutator.webhookNamespace)
		err := initGenerator.GenerateForNamespace(request.Context, request.DynaKube, request.Namespace.Name)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Info("failed to create the init secret before oneagent pod injection")
			return err
		}
		log.Info("ensured that the init secret is present before oneagent pod injection")
	} else if err != nil {
		log.Info("failed to query the init secret before oneagent pod injection")
		return errors.WithStack(err)
	}
	return nil
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
		return string(consts.AgentCsiMode)
	}
	return string(consts.AgentInstallerMode)
}
