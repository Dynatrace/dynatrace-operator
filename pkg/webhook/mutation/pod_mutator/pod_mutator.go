package pod_mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	kubeobjects2 "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ocDebugAnnotationsContainer = "debug.openshift.io/source-container"
	ocDebugAnnotationsResource  = "debug.openshift.io/source-resource"
)

// AddPodMutationWebhookToManager adds the Webhook server to the Manager
func AddPodMutationWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv(kubeobjects2.EnvPodName)
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
		return err
	}
	registerLivezEndpoint(mgr)
	return nil
}

// podMutatorWebhook executes mutators on Pods
type podMutatorWebhook struct {
	apiReader client.Reader
	decoder   admission.Decoder
	recorder  podMutatorEventRecorder

	webhookImage     string
	webhookNamespace string
	clusterID        string
	apmExists        bool
	deployedViaOLM   bool

	mutators []dtwebhook.PodMutator
}

func (webhook *podMutatorWebhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	emptyPatch := admission.Patched("")
	mutationRequest, err := webhook.createMutationRequestBase(ctx, request)
	if err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}
	if !mutationRequired(mutationRequest) || webhook.isOcDebugPod(mutationRequest.Pod) {
		return emptyPatch
	}

	podName := mutationRequest.PodName()
	webhook.setupEventRecorder(mutationRequest)

	if webhook.isInjected(mutationRequest) {
		if webhook.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", podName)
			webhook.recorder.sendPodUpdateEvent()
			return createResponseForPod(mutationRequest.Pod, request)
		}
		log.Info("no change, all containers already injected", "podName", podName)
		return emptyPatch
	}

	if err := webhook.handlePodMutation(mutationRequest); err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}
	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}
	return kubeobjects2.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)
}

func (webhook *podMutatorWebhook) setupEventRecorder(mutationRequest *dtwebhook.MutationRequest) {
	webhook.recorder.dynakube = &mutationRequest.DynaKube
	webhook.recorder.pod = mutationRequest.Pod
}

func (webhook *podMutatorWebhook) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	for _, mutator := range webhook.mutators {
		if mutator.Injected(mutationRequest.BaseRequest) {
			return true
		}
	}
	return false
}

func (webhook *podMutatorWebhook) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}

func (webhook *podMutatorWebhook) handlePodMutation(mutationRequest *dtwebhook.MutationRequest) error {
	mutationRequest.InstallContainer = createInstallInitContainerBase(webhook.webhookImage, webhook.clusterID, mutationRequest.Pod, mutationRequest.DynaKube)
	isMutated := false
	for _, mutator := range webhook.mutators {
		if !mutator.Enabled(mutationRequest.BaseRequest) {
			continue
		}
		if err := mutator.Mutate(mutationRequest); err != nil {
			return err
		}
		isMutated = true
	}
	if !isMutated {
		log.Info("no mutation is enabled")
		return nil
	}

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	webhook.recorder.sendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)
	return nil
}

func (webhook *podMutatorWebhook) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool

	if mutationRequest.DynaKube.FeatureDisableWebhookReinvocationPolicy() {
		return false
	}

	reinvocationRequest := mutationRequest.ToReinvocationRequest()
	for _, mutator := range webhook.mutators {
		if mutator.Enabled(mutationRequest.BaseRequest) {
			if update := mutator.Reinvoke(reinvocationRequest); update {
				needsUpdate = true
			}
		}
	}
	return needsUpdate
}

func setDynatraceInjectedAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}
	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
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
	podName := kubeobjects2.GetPodName(*pod)
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())
	return rsp
}
