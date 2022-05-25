package oneagent_mutation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
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

func NewOneAgentPodMutator(image, clusterID, webhookNamespace string, client client.Client, apiReader client.Reader) *OneAgentPodMutator {
	return &OneAgentPodMutator{
		image:            image,
		clusterID:        clusterID,
		webhookNamespace: webhookNamespace,
		client:           client,
		apiReader:        apiReader,
	}
}

func (mutator *OneAgentPodMutator) Enabled(pod *corev1.Pod) bool {
	return kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInject, true)
}

func (mutator *OneAgentPodMutator) Injected(pod *corev1.Pod) bool {
	return kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInjected, false)
}

func (mutator *OneAgentPodMutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Info("injecting OneAgent into pod", "pod", request.Pod.GenerateName)
	if err := mutator.ensureInitSecret(request); err != nil {
		return err
	}

	installerInfo := getInstallerInfo(request.Pod)
	mutator.addVolumes(request.Pod, request.DynaKube)
	mutator.configureInitContainer(request, installerInfo)
	mutator.mutateUserContainers(request)
	setInjectedAnnotation(request.Pod)
	return nil
}

func (mutator *OneAgentPodMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mutator.Injected(request.Pod) {
		return false
	}
	log.Info("reinvoking", "pod", request.Pod.GenerateName)
	mutator.reinvokeUserContainers(request)
	return true
}

func (mutator *OneAgentPodMutator) getVolumeMode(dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.NeedsCSIDriver() {
		return provisionedVolumeMode
	}
	return installerVolumeMode
}

func (mutator *OneAgentPodMutator) ensureInitSecret(request *dtwebhook.MutationRequest) error {
	var initSecret corev1.Secret
	secretObjectKey := client.ObjectKey{Name: dtwebhook.SecretConfigName, Namespace: request.Namespace.Name}
	if err := mutator.apiReader.Get(request.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
		initGenerator := initgeneration.NewInitGenerator(mutator.client, mutator.apiReader, mutator.webhookNamespace)
		_, err := initGenerator.GenerateForNamespace(request.Context, *request.DynaKube, request.Namespace.Name)
		if err != nil {
			log.Error(err, "Failed to create the init secret before oneagent pod injection")
			return err
		}
		log.Info("created the init secret before oneagent pod injection")
	} else if err != nil {
		log.Error(err, "failed to query the init secret before oneagent pod injection")
		return err
	}
	return nil
}

func containerIsInjected(container *corev1.Container) bool {
	for _, e := range container.Env {
		if e.Name == "LD_PRELOAD" {
			return true
		}
	}
	return false
}
