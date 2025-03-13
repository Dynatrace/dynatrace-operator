package metadata

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	client           client.Client
	metaClient       client.Client
	apiReader        client.Reader
	webhookNamespace string
}

func NewMutator(webhookNamespace string, client client.Client, apiReader client.Reader, metaClient client.Client) *Mutator {
	return &Mutator{
		client:           client,
		apiReader:        apiReader,
		metaClient:       metaClient,
		webhookNamespace: webhookNamespace,
	}
}

func (mut *Mutator) Enabled(request *dtwebhook.BaseRequest) bool {
	return metacommon.IsEnabled(request)
}

func (mut *Mutator) Injected(request *dtwebhook.BaseRequest) bool {
	return metacommon.IsInjected(request)
}

func (mut *Mutator) Mutate(ctx context.Context, request *dtwebhook.MutationRequest) error {
	log.Info("injecting metadata-enrichment into pod", "podName", request.PodName())

	workload, err := metacommon.RetrieveWorkload(mut.metaClient, request)
	if err != nil {
		return err
	}

	err = mut.ensureIngestEndpointSecret(request)
	if err != nil {
		return err
	}

	setupVolumes(request.Pod)
	mutateUserContainers(request.BaseRequest)
	updateInstallContainer(request.InstallContainer, workload, request.DynaKube.Status.KubernetesClusterName, request.DynaKube.Status.KubernetesClusterMEID)
	propagateMetadataAnnotations(request)
	metacommon.SetInjectedAnnotation(request.Pod)
	metacommon.SetWorkloadAnnotations(request.Pod, workload)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mut.Injected(request.BaseRequest) {
		return false
	}

	log.Info("reinvoking", "podName", request.PodName())

	return reinvokeUserContainers(request.BaseRequest)
}

func (mut *Mutator) ensureIngestEndpointSecret(request *dtwebhook.MutationRequest) error {
	endpointGenerator := dtingestendpoint.NewSecretGenerator(mut.client, mut.apiReader, mut.webhookNamespace)

	var endpointSecret corev1.Secret

	err := mut.apiReader.Get(
		request.Context,
		client.ObjectKey{
			Name:      consts.EnrichmentEndpointSecretName,
			Namespace: request.Namespace.Name,
		},
		&endpointSecret)
	if k8serrors.IsNotFound(err) {
		err := endpointGenerator.GenerateForNamespace(request.Context, request.DynaKube.Name, request.Namespace.Name)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Info("failed to create the ingest endpoint secret before pod injection")

			return err
		}

		log.Info("ensured that the ingest endpoint secret is present before pod injection")
	} else if err != nil {
		log.Info("failed to query the ingest endpoint secret before pod injection")

		return err
	}

	return nil
}

func ContainerIsInjected(container corev1.Container) bool {
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name == workloadEnrichmentVolumeName || volumeMount.Name == ingestEndpointVolumeName {
			return true
		}
	}

	return false
}
