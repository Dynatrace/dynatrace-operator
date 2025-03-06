package v1

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/metadata"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/oneagent"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Injector struct {
	recorder            events.EventRecorder
	isContainerInjected func(corev1.Container) bool
	webhookImage        string
	clusterID           string

	mutators []dtwebhook.PodMutator
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(apiReader client.Reader, kubeClient, metaClient client.Client, recorder events.EventRecorder, clusterID, webhookPodImage, webhookNamespace string) *Injector { //nolint:revive
	return &Injector{
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
		isContainerInjected: containerIsInjected,
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
	} else {
		if err := wh.handlePodMutation(ctx, mutationRequest); err != nil {
			return err
		}
	}

	log.Info("injection finished for pod", "podName", mutationRequest.PodName, "namespace", mutationRequest.Namespace)

	return nil
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

func (wh *Injector) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
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
