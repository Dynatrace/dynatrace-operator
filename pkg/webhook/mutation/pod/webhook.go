package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	podv2 "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ocDebugAnnotationsContainer = "debug.openshift.io/source-container"
	ocDebugAnnotationsResource  = "debug.openshift.io/source-resource"
)

func AddWebhookToManager(ctx context.Context, mgr manager.Manager, ns string) error {
	podName := os.Getenv(env.PodName)
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(ctx, mgr, ns, podName); err != nil {
		return err
	}

	registerLivezEndpoint(mgr)

	return nil
}

type webhook struct {
	v1 dtwebhook.PodInjector
	v2 dtwebhook.PodInjector

	recorder events.EventRecorder
	decoder  admission.Decoder

	apiReader client.Reader

	webhookNamespace string
	deployedViaOLM   bool
}

func (wh *webhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	emptyPatch := admission.Patched("")
	mutationRequest, err := wh.createMutationRequestBase(ctx, request)

	if err != nil {
		emptyPatch.Result.Message = fmt.Sprintf("unable to inject into pod (err=%s)", err.Error())
		log.Error(err, "building mutation request base encountered an error")

		return emptyPatch
	}

	if mutationRequest == nil {
		emptyPatch.Result.Message = "injection into pod not required"

		return emptyPatch
	}

	podName := mutationRequest.PodName()

	if !mutationRequired(mutationRequest) || wh.isOcDebugPod(mutationRequest.Pod) {
		return emptyPatch
	}

	wh.recorder.Setup(mutationRequest)

	if podv2.IsEnabled(mutationRequest) {
		err := wh.v2.Handle(ctx, mutationRequest)
		if err != nil {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	} else {
		err := wh.v1.Handle(ctx, mutationRequest)
		if err != nil {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	}

	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}

	enabledOnPod := maputils.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)

	enabledOnContainers := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		enabledOnContainers = enabledOnContainers || !dtwebhook.IsContainerExcludedFromInjection(mutationRequest.DynaKube.Annotations, mutationRequest.Pod.Annotations, container.Name)
	}

	return enabledOnPod && enabledOnContainers
}

func (wh *webhook) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}

// createResponseForPod tries to format pod as json
func createResponseForPod(pod *corev1.Pod, req admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return silentErrorResponse(pod, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func silentErrorResponse(pod *corev1.Pod, err error) admission.Response {
	rsp := admission.Patched("")
	podName := k8spod.GetName(*pod)
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())

	return rsp
}
