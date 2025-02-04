package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
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
	decoder  admission.Decoder
	recorder eventRecorder

	apiReader client.Reader

	webhookImage     string
	webhookNamespace string
	clusterID        string

	mutators       []dtwebhook.PodMutator
	apmExists      bool
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

	wh.setupEventRecorder(mutationRequest)

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", podName)
			wh.recorder.sendPodUpdateEvent()

			return createResponseForPod(mutationRequest.Pod, request)
		}

		log.Info("no change, all containers already injected", "podName", podName)

		return emptyPatch
	}

	if err := wh.handlePodMutation(ctx, mutationRequest); err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}

	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}

	return maputils.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)
}

func (wh *webhook) setupEventRecorder(mutationRequest *dtwebhook.MutationRequest) {
	wh.recorder.dk = &mutationRequest.DynaKube
	wh.recorder.pod = mutationRequest.Pod
}

func (wh *webhook) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	for _, mutator := range wh.mutators {
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

func (wh *webhook) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}

func podNeedsInjection(mutationRequest *dtwebhook.MutationRequest) bool {
	needsInjection := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		needsInjection = needsInjection || !dtwebhook.IsContainerExcludedFromInjection(mutationRequest.DynaKube.Annotations, mutationRequest.Pod.Annotations, container.Name)
	}

	return needsInjection
}

func (wh *webhook) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	if !podNeedsInjection(mutationRequest) {
		log.Info("no mutation is needed, all containers are excluded from injection.")

		return nil
	}

	mutationRequest.InstallContainer = createInstallInitContainerBase(wh.webhookImage, wh.clusterID, mutationRequest.Pod, mutationRequest.DynaKube)

	_ = updateContainerInfo(mutationRequest.BaseRequest, mutationRequest.InstallContainer)

	var isMutated bool

	for _, mutator := range wh.mutators {
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
	wh.recorder.sendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)

	return nil
}

func (wh *webhook) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	var needsUpdate bool

	reinvocationRequest := mutationRequest.ToReinvocationRequest()

	isMutated := updateContainerInfo(reinvocationRequest.BaseRequest, nil)

	if !isMutated { // == no new containers were detected, we only mutate new containers during reinvoke
		return false
	}

	for _, mutator := range wh.mutators {
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
