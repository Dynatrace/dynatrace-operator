package v2

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/oneagent"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Injector struct {
	recorder events.EventRecorder

	mutators []dtwebhook.PodMutator
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(apiReader client.Reader, recorder events.EventRecorder) *Injector {
	return &Injector{
		recorder: recorder,
		mutators: []dtwebhook.PodMutator{
			oamutation.NewMutator(apiReader),
		},
	}
}

func (wh *Injector) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName())
			wh.recorder.SendPodUpdateEvent()
		}

		log.Info("no change, all containers already injected", "podName", mutationRequest.PodName())
	} else {
		if err := wh.handlePodMutation(ctx, mutationRequest); err != nil {
			return err
		}
	}

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

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
	var err error

	mutationRequest.InstallContainer, err = createInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)
	if err != nil {
		log.Error(err, "failed to create init container")

		return err
	}

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

	for _, mutator := range wh.mutators {
		if mutator.Enabled(mutationRequest.BaseRequest) {
			if update := mutator.Reinvoke(reinvocationRequest); update {
				needsUpdate = true
			}
		}
	}

	return needsUpdate
}
