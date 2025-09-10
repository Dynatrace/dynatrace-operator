package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ocDebugAnnotationsContainer = "debug.openshift.io/source-container"
	ocDebugAnnotationsResource  = "debug.openshift.io/source-resource"
)

func AddWebhookToManager(ctx context.Context, mgr manager.Manager, ns string, isOpenShift bool) error {
	podName := os.Getenv(env.PodName)
	if podName == "" {
		log.Info("no Pod name set for dtInjectionWebhook container")
	}

	if err := registerInjectEndpoint(ctx, mgr, ns, podName, isOpenShift); err != nil {
		return err
	}

	registerLivezEndpoint(mgr)

	return nil
}

type dtInjectionWebhook struct {
	webhookBase
	metaMutator dtwebhook.Mutator
	oaMutator   dtwebhook.Mutator

	webhookPodImage string
	isOpenShift     bool

	handleFunc func(admission.Request) admission.Response
}

func (wh *dtInjectionWebhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	emptyPatch := admission.Patched("")

	mutationRequest := wh.preparePodMutationRequest(ctx, &emptyPatch, request)
	if mutationRequest == nil {
		return emptyPatch
	}

	// TODO check if we need separate endpoint for otlp injection
	err := wh.handle(mutationRequest)
	if err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func MutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
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
