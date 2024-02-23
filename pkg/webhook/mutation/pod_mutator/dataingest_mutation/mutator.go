package dataingest_mutation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/internal/otel"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataIngestPodMutator struct {
	client           client.Client
	metaClient       client.Client
	apiReader        client.Reader
	webhookNamespace string
}

func NewDataIngestPodMutator(webhookNamespace string, client client.Client, apiReader client.Reader, metaClient client.Client) *DataIngestPodMutator {
	return &DataIngestPodMutator{
		client:           client,
		apiReader:        apiReader,
		metaClient:       metaClient,
		webhookNamespace: webhookNamespace,
	}
}

func (mutator *DataIngestPodMutator) Enabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationDataIngestInject,
		request.DynaKube.FeatureAutomaticInjection())
	enabledOnDynakube := !request.DynaKube.FeatureDisableMetadataEnrichment()

	return enabledOnPod && enabledOnDynakube
}

func (mutator *DataIngestPodMutator) Injected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationDataIngestInjected, false)
}

func (mutator *DataIngestPodMutator) Mutate(ctx context.Context, request *dtwebhook.MutationRequest) error {
	_, span := dtotel.StartSpan(ctx, webhookotel.Tracer())
	defer span.End()

	log.Info("injecting data-ingest into pod", "podName", request.PodName())

	workload, err := mutator.retrieveWorkload(request)
	if err != nil {
		span.RecordError(err)

		return err
	}

	err = mutator.ensureDataIngestSecret(request)
	if err != nil {
		span.RecordError(err)

		return err
	}

	setupVolumes(request.Pod)
	mutateUserContainers(request.BaseRequest)
	updateInstallContainer(request.InstallContainer, workload)
	setInjectedAnnotation(request.Pod)
	setWorkloadAnnotations(request.Pod, workload)

	return nil
}

func (mutator *DataIngestPodMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mutator.Injected(request.BaseRequest) {
		return false
	}

	log.Info("reinvoking", "podName", request.PodName())

	return reinvokeUserContainers(request.BaseRequest)
}

func (mutator *DataIngestPodMutator) ensureDataIngestSecret(request *dtwebhook.MutationRequest) error {
	endpointGenerator := dtingestendpoint.NewEndpointSecretGenerator(mutator.client, mutator.apiReader, mutator.webhookNamespace)

	var endpointSecret corev1.Secret

	err := mutator.apiReader.Get(
		request.Context,
		client.ObjectKey{
			Name:      consts.EnrichmentEndpointSecretName,
			Namespace: request.Namespace.Name,
		},
		&endpointSecret)
	if k8serrors.IsNotFound(err) {
		err := endpointGenerator.GenerateForNamespace(request.Context, request.DynaKube.Name, request.Namespace.Name)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Info("failed to create the data-ingest endpoint secret before pod injection")

			return err
		}

		log.Info("ensured that the data-ingest endpoint secret is present before pod injection")
	} else if err != nil {
		log.Info("failed to query the data-ingest endpoint secret before pod injection")

		return err
	}

	return nil
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[dtwebhook.AnnotationDataIngestInjected] = "true"
}

func setWorkloadAnnotations(pod *corev1.Pod, workload *workloadInfo) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[dtwebhook.AnnotationWorkloadKind] = workload.kind
	pod.Annotations[dtwebhook.AnnotationWorkloadName] = workload.name
}

func containerIsInjected(container *corev1.Container) bool {
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name == workloadEnrichmentVolumeName || volumeMount.Name == ingestEndpointVolumeName {
			return true
		}
	}

	return false
}
