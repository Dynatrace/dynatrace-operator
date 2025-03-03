package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/metadata"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/oneagent"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Injector struct {
	recorder     events.EventRecorder
	webhookImage string
	clusterID    string

	mutators []dtwebhook.PodMutator
}

func NewInjector(apiReader client.Reader, kubeClient, metaClient client.Client, recorder events.EventRecorder, clusterID, webhookPodImage, webhookNamespace string) Injector {
	return Injector{
		webhookImage: webhookPodImage,
		recorder:     recorder,
		clusterID:    clusterID,
		mutators: []dtwebhook.PodMutator{
			oamutation.NewMutator(
				clusterID,
				webhookNamespace,
				kubeClient,
				apiReader,
			),
			metadata.NewMutator(
				webhookNamespace,
				kubeClient,
				apiReader,
				metaClient,
			),
		},
	}
}

func (wh *Injector) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName)
			wh.recorder.SendPodUpdateEvent()
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

func (wh *Injector) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
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

func podNeedsInjection(mutationRequest *dtwebhook.MutationRequest) bool {
	needsInjection := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		needsInjection = needsInjection || !dtwebhook.IsContainerExcludedFromInjection(mutationRequest.DynaKube.Annotations, mutationRequest.Pod.Annotations, container.Name)
	}

	return needsInjection
}

func (wh *Injector) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
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
	wh.recorder.SendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)

	return nil
}

func (wh *Injector) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
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
