package pod

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/otlp"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type otlpInjectionWebhook struct {
	webhookBase
	otlpMutator dtwebhook.Mutator
}

func newOtlpInjectionWebhook( //nolint:revive
	kubeClient client.Client,
	apiReader client.Reader,
	eventRecorder events.EventRecorder,
	decoder admission.Decoder,
	webhookPod corev1.Pod) *otlpInjectionWebhook {
	return &otlpInjectionWebhook{
		webhookBase: webhookBase{
			kubeClient:       kubeClient,
			decoder:          decoder,
			apiReader:        apiReader,
			webhookNamespace: webhookPod.Namespace,
			deployedViaOLM:   kubesystem.IsDeployedViaOlm(webhookPod),
			recorder:         eventRecorder,
		},
		otlpMutator: otlp.NewMutator(),
	}
}

func (wh *otlpInjectionWebhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	emptyPatch := admission.Patched("")

	mutationRequest := wh.preparePodMutationRequest(ctx, &emptyPatch, request)
	if mutationRequest == nil {
		return emptyPatch
	}

	err := wh.handle(mutationRequest)
	if err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func (wh *otlpInjectionWebhook) handle(mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if !wh.otlpMutator.IsEnabled(mutationRequest.BaseRequest) {
		return nil
	}

	if wh.otlpMutator.IsInjected(mutationRequest.BaseRequest) {
		// handle reinvocation
		wh.otlpMutator.Reinvoke(mutationRequest.ToReinvocationRequest())
		return nil
	}

	return wh.otlpMutator.Mutate(mutationRequest)
}
