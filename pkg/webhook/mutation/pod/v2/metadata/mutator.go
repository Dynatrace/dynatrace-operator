package metadata

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	corev1 "k8s.io/api/core/v1"
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

	// TODO

	metacommon.SetInjectedAnnotation(request.Pod)
	metacommon.SetWorkloadAnnotations(request.Pod, workload)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mut.Injected(request.BaseRequest) {
		return false
	}

	// TODO

	return true
}

func ContainerIsInjected(container corev1.Container) bool {
	return true // TODO
}
