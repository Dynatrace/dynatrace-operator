package dataingest_mutation

import (
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataIngestPodMutator struct {
	webhookNamespace string
	client           client.Client
	metaClient       client.Client
	apiReader        client.Reader
}

func NewDataIngestPodMutator(webhookNamespace string, client client.Client, apiReader client.Reader, metaClient client.Client) *DataIngestPodMutator {
	return &DataIngestPodMutator{
		client:           client,
		apiReader:        apiReader,
		metaClient:       metaClient,
		webhookNamespace: webhookNamespace,
	}
}

func (mutator *DataIngestPodMutator) Enabled(pod *corev1.Pod) bool {
	return kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInject, true)
}

func (mutator *DataIngestPodMutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Info("injecting data-ingest into pod", "pod", request.Pod.GenerateName)
	workload, err := mutator.retrieveWorkload(request)
	if err != nil {
		return err
	}
	err = mutator.ensureDataIngestSecret(request)
	if err != nil {
		return err
	}
	mutator.setupVolumes(request.Pod)
	mutator.updateContainers(request.Pod)
	updateInstallContainer(request.InitContainer, workload)
	for _, container := range request.Pod.Spec.Containers {
		setupVolumeMounts(&container)
	}
	setAnnotation(request.Pod)
	return nil
}

func (mutator *DataIngestPodMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	var needsUpdate = false
	if !podIsInjected(request.Pod) {
		return false
	}
	log.Info("reinvoking", "pod", request.Pod.GenerateName)

	for _, container := range request.Pod.Spec.Containers {
		if containerIsInjected(container) {
			continue
		}
		setupVolumeMounts(&container)
		needsUpdate = true
	}
	return needsUpdate
}

func (mutator *DataIngestPodMutator) ensureDataIngestSecret(request *dtwebhook.MutationRequest) error {
	endpointGenerator := dtingestendpoint.NewEndpointSecretGenerator(mutator.client, mutator.apiReader, mutator.webhookNamespace)

	var endpointSecret corev1.Secret
	if err := mutator.apiReader.Get(request.Context, client.ObjectKey{Name: dtingestendpoint.SecretEndpointName, Namespace: request.Namespace.Name}, &endpointSecret); k8serrors.IsNotFound(err) {
		if _, err := endpointGenerator.GenerateForNamespace(request.Context, request.DynaKube.Name, request.Namespace.Name); err != nil {
			log.Error(err, "failed to create the data-ingest endpoint secret before pod injection")
			return err
		}
		log.Info("created the data-ingest endpoint secret before pod injection")
	} else if err != nil {
		log.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return err
	}

	return nil
}

func setAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[dtwebhook.AnnotationDataIngestInjected] = "true"
}

func podIsInjected(pod *corev1.Pod) bool {
	return kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInjected, false)
}

func containerIsInjected(container corev1.Container) bool {
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name == EnrichmentVolumeName || volumeMount.Name == EnrichmentEndpointVolumeName {
			return true
		}
	}
	return false
}
