package pod

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type webhookBase struct {
	recorder         events.EventRecorder
	decoder          admission.Decoder
	kubeClient       client.Client
	apiReader        client.Reader
	webhookNamespace string

	deployedViaOLM bool
}

func (wh *webhookBase) preparePodMutationRequest(ctx context.Context, emptyPatch *admission.Response, request admission.Request) *mutator.MutationRequest {
	mutationRequest, err := wh.createMutationRequestBase(ctx, request)
	if err != nil {
		emptyPatch.Result.Message = fmt.Sprintf("unable to inject into pod (err=%s)", err.Error())
		log.Error(err, "building mutation request base encountered an error")

		return nil
	}

	if mutationRequest == nil {
		emptyPatch.Result.Message = "injection into pod not required"

		return nil
	}

	if !MutationRequired(mutationRequest) || wh.isOcDebugPod(mutationRequest.Pod) {
		return nil
	}

	wh.recorder.Setup(mutationRequest)
	return mutationRequest
}

func (wh *webhookBase) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}
