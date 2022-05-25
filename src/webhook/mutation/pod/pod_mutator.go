package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AddPodMutationWebhookToManager adds the Webhook server to the Manager
func AddPodMutationWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
		return err
	}
	registerLivezEndpoint(mgr)
	return nil
}

// podMutatorWebhook injects the OneAgent into Pods
type podMutatorWebhook struct {
	apiReader client.Reader
	decoder   *admission.Decoder
	recorder  podMutatorEventRecorder

	webhookImage     string
	webhookNamespace string
	clusterID        string
	apmExists        bool

	mutators []dtwebhook.PodMutator
}

// InjectDecoder injects the decoder
func (webhook *podMutatorWebhook) InjectDecoder(d *admission.Decoder) error {
	webhook.decoder = d
	return nil
}

func (webhook *podMutatorWebhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	emptyPatch := admission.Patched("")

	if webhook.apmExists {
		return emptyPatch
	}

	mutationRequest, err := webhook.createMutationRequestBase(ctx, request)
	if err != nil {
		return silentErrorResponse(mutationRequest.Pod.Name, err)
	}

	webhook.setupEventRecorder(mutationRequest)

	if webhook.isInjected(mutationRequest) {
		if webhook.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.Pod.GenerateName)
			webhook.recorder.sendPodUpdateEvent()
			return createResponseForPod(mutationRequest.Pod, request)
		}
		log.Info("pod already injected, no change", "podName", mutationRequest.Pod.GenerateName)
		return emptyPatch
	}

	if err := webhook.handlePodMutation(mutationRequest); err != nil {
		log.Error(err, "failed to inject into pod")
		return silentErrorResponse(mutationRequest.Pod.GenerateName, err)
	}
	log.Info("injection finished for pod", "podName", mutationRequest.Pod.GenerateName, "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func (webhook *podMutatorWebhook) setupEventRecorder(mutationRequest *dtwebhook.MutationRequest) {
	webhook.recorder.dynakube = mutationRequest.DynaKube
	webhook.recorder.pod = mutationRequest.Pod
}

func (webhook *podMutatorWebhook) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	for _, mutator := range webhook.mutators {
		if mutator.Injected(mutationRequest.Pod) {
			return true
		}
	}
	return false
}

func (webhook *podMutatorWebhook) handlePodMutation(mutationRequest *dtwebhook.MutationRequest) error {
	mutationRequest.InstallContainer = webhook.createInstallInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)
	for _, mutator := range webhook.mutators {
		if !mutator.Enabled(mutationRequest.Pod) {
			continue
		}
		if err := mutator.Mutate(mutationRequest); err != nil {
			return err
		}
	}
	addToInitContainers(mutationRequest.Pod, mutationRequest.InstallContainer)
	webhook.recorder.sendPodInjectEvent()
	setAnnotation(mutationRequest)
	return nil
}

func (webhook *podMutatorWebhook) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool

	if !mutationRequest.DynaKube.FeatureEnableWebhookReinvocationPolicy() {
		return false
	}

	reinvocationRequest := mutationRequest.ToReinvocationRequest()
	for _, mutator := range webhook.mutators {
		if mutator.Enabled(mutationRequest.Pod) {
			if update := mutator.Reinvoke(reinvocationRequest); update {
				needsUpdate = true
			}
		}
	}
	return needsUpdate
}

func setAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}
	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
}

// createResponseForPod tries to format pod as json
func createResponseForPod(pod *corev1.Pod, req admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return silentErrorResponse(pod.Name, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func silentErrorResponse(podName string, err error) admission.Response {
	rsp := admission.Patched("")
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())
	return rsp
}
