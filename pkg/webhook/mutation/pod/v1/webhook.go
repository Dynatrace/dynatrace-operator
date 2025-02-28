package pod

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Webhook struct {
	Recorder events.EventRecorder

	ApiReader client.Reader

	WebhookImage     string
	WebhookNamespace string
	ClusterID        string

	Mutators []dtwebhook.PodMutator
}

func (wh *Webhook) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	wh.Recorder.Setup(mutationRequest)

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName)
			wh.Recorder.SendPodUpdateEvent()
		}

		log.Info("no change, all containers already injected", "podName", mutationRequest.PodName)
	}

	if err := wh.handlePodMutation(ctx, mutationRequest); err != nil {
		return err
	}

	log.Info("injection finished for pod", "podName", mutationRequest.PodName, "namespace", mutationRequest.Namespace)

	return nil
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}

	return maputils.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)
}

func (wh *Webhook) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	for _, mutator := range wh.Mutators {
		if mutator.Injected(mutationRequest.BaseRequest) {
			return true
		}
	}

	installContainer := container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)
	if installContainer != nil {
		log.Info("Dynatrace init-container already present, skipping mutation, doing reinvocation", "containerName", dtwebhook.InstallContainerName)

		return true
	}

	return false
}

func podNeedsInjection(mutationRequest *dtwebhook.MutationRequest) bool {
	needsInjection := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		needsInjection = needsInjection || !dtwebhook.IsContainerExcludedFromInjection(mutationRequest.DynaKube.Annotations, mutationRequest.Pod.Annotations, container.Name)
	}

	return needsInjection
}

func (wh *Webhook) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	if !podNeedsInjection(mutationRequest) {
		log.Info("no mutation is needed, all containers are excluded from injection.")

		return nil
	}

	mutationRequest.InstallContainer = createInstallInitContainerBase(wh.WebhookImage, wh.ClusterID, mutationRequest.Pod, mutationRequest.DynaKube)

	_ = updateContainerInfo(mutationRequest.BaseRequest, mutationRequest.InstallContainer)

	var isMutated bool

	for _, mutator := range wh.Mutators {
		if !mutator.Enabled(mutationRequest.BaseRequest) {
			continue
		}

		if err := mutator.Mutate(ctx, mutationRequest); err != nil {
			return err
		}

		isMutated = true
	}

	if !isMutated {
		log.Info("no mutation is enabled")

		return nil
	}

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	wh.Recorder.SendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)

	return nil
}

func (wh *Webhook) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool

	reinvocationRequest := mutationRequest.ToReinvocationRequest()

	isMutated := updateContainerInfo(reinvocationRequest.BaseRequest, nil)

	if !isMutated { // == no new containers were detected, we only mutate new containers during reinvoke
		return false
	}

	for _, mutator := range wh.Mutators {
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
	podName := k8spod.GetName(*pod)
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())

	return rsp
}
