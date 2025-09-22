package pod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	log = logd.Get().WithName("pod-mutation")
)

const (
	ocDebugAnnotationsContainer = "debug.openshift.io/source-container"
	ocDebugAnnotationsResource  = "debug.openshift.io/source-resource"
)

func AddWebhookToManager(ctx context.Context, mgr manager.Manager, ns string, isOpenShift bool) error {
	podName := os.Getenv(env.PodName)
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(ctx, mgr, ns, podName, isOpenShift); err != nil {
		return err
	}

	registerLivezEndpoint(mgr)

	return nil
}

type webhook struct {
	recorder events.EventRecorder

	injectionHandler handler.Handler
	otlpHandler      handler.Handler

	decoder admission.Decoder

	kubeClient client.Client
	apiReader  client.Reader

	webhookPodImage  string
	webhookNamespace string
	isOpenShift      bool

	deployedViaOLM bool
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

	originalPod := mutationRequest.Pod.DeepCopy()

	var mutErr *dtwebhook.MutatorError
	if err := wh.injectionHandler.Handle(mutationRequest); err != nil {
		mutErr = new(dtwebhook.MutatorError)
		if !errors.As(err, mutErr) {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	} else if err := wh.otlpHandler.Handle(mutationRequest); err != nil {
		mutErr = new(dtwebhook.MutatorError)
		if !errors.As(err, mutErr) {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	}

	if mutErr != nil {
		mutationRequest.Pod = originalPod // prevent partial modifications
		mutErr.SetAnnotations(mutationRequest.Pod)
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
