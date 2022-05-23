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

	if !webhook.handleAlreadyInjectedPod(mutationRequest) {
		if err := webhook.handleFreshPod(mutationRequest); err != nil {
			log.Error(err, "failed to inject into pod")
			return silentErrorResponse(mutationRequest.Pod.GenerateName, err)
		}
		log.Info("injecting into Pod", "generatedName", mutationRequest.Pod.GenerateName, "namespace", request.Namespace)
	}
	return createResponseForPod(mutationRequest.Pod, request)
}

func (webhook *podMutatorWebhook) setupEventRecorder(mutationRequest *dtwebhook.MutationRequest) {
	webhook.recorder.dynakube = mutationRequest.DynaKube
	webhook.recorder.pod = mutationRequest.Pod
}

func (webhook *podMutatorWebhook) handleFreshPod(mutationRequest *dtwebhook.MutationRequest) error {
	mutationRequest.InitContainer = webhook.createInstallInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)

	for _, mutator := range webhook.mutators {
		if !mutator.Enabled(mutationRequest.Pod) {
			continue
		}
		if err := mutator.Mutate(mutationRequest); err != nil {
			return err
		}
	}
	addToInitContainers(mutationRequest.Pod, mutationRequest.InitContainer)
	webhook.recorder.sendPodInjectEvent()
	setAnnotation(mutationRequest)
	return nil
}

func (webhook *podMutatorWebhook) handleAlreadyInjectedPod(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool
	if mutationRequest.DynaKube.FeatureEnableWebhookReinvocationPolicy() {
		if webhook.applyReinvocationPolicy(mutationRequest) {
			log.Info("updating pod with missing containers")
			webhook.recorder.sendPodUpdateEvent()
			needsUpdate = true
		}
	}
	return needsUpdate
}

func (webhook *podMutatorWebhook) applyReinvocationPolicy(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool
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
