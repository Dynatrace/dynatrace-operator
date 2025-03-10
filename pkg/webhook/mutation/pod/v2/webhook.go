package v2

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/oneagent"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Injector struct {
	recorder  events.EventRecorder
	apiReader client.Reader
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(apiReader client.Reader, recorder events.EventRecorder) *Injector {
	return &Injector{
		recorder:  recorder,
		apiReader: apiReader,
	}
}

func (wh *Injector) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if !wh.isInputSecretPresent(mutationRequest) {
		return nil
	}

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

	setDynatraceInjectedAnnotation(mutationRequest)

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (wh *Injector) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
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

	updated := oamutation.Mutate(mutationRequest)
	if !updated {
		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoMutationNeeded)

		return nil
	}

	// TODO: Add `--attribute-container` for new containers to init-container
	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	wh.recorder.SendPodInjectEvent()

	return nil
}

func (wh *Injector) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	updated := oamutation.Reinvoke(mutationRequest.BaseRequest)
	// TODO: Add `--attribute-container` for new containers to init-container

	return updated
}

func (wh *Injector) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest) bool {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: mutationRequest.Namespace.Name}
	if err := wh.apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
		log.Info("dynatrace-bootstrapper-config is not available, injection not possible", "pod", mutationRequest.PodName())

		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoBootstrapperConfigReason)

		return false
	}

	return true
}

func setDynatraceInjectedAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
	delete(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceReason)
}
