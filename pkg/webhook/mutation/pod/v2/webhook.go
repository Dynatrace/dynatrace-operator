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
	recorder   events.EventRecorder
	apiReader  client.Reader
	metaClient client.Client
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder) *Injector {
	return &Injector{
		recorder:   recorder,
		apiReader:  apiReader,
		metaClient: metaClient,
	}
}

func (wh *Injector) Handle(_ context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if !wh.isInputSecretPresent(mutationRequest) {
		return nil
	}

	if !isCustomImageSet(mutationRequest) {
		return nil
	}

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName())
			wh.recorder.SendPodUpdateEvent()

			return nil
		}

		log.Info("no change, all containers already injected", "podName", mutationRequest.PodName())
	} else {
		if err := wh.handlePodMutation(mutationRequest); err != nil {
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

func (wh *Injector) handlePodMutation(mutationRequest *dtwebhook.MutationRequest) error {
	mutationRequest.InstallContainer = createInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)

	addContainerAttributes(mutationRequest)
	wh.addPodAttributes(mutationRequest)

	updated := oamutation.Mutate(mutationRequest)
	if !updated {
		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoMutationNeededReason)

		return nil
	}

	oacommon.SetInjectedAnnotation(mutationRequest.Pod)

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	wh.recorder.SendPodInjectEvent()

	return nil
}

func (wh *Injector) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	mutationRequest.InstallContainer = container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)
	addContainerAttributes(mutationRequest)

	updated := oamutation.Reinvoke(mutationRequest.BaseRequest)

	return updated
}

func isCustomImageSet(mutationRequest *dtwebhook.MutationRequest) bool {
	customImage := mutationRequest.DynaKube.OneAgent().GetCustomCodeModulesImage()
	if customImage == "" {
		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoCodeModulesImageReason)

		return false
	}

	return true
}

func (wh *Injector) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest) bool {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: mutationRequest.Namespace.Name}
	err := wh.apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret)

	if k8serrors.IsNotFound(err) {
		log.Info("dynatrace-bootstrapper-config is not available, injection not possible", "pod", mutationRequest.PodName())

		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoBootstrapperConfigReason)

		return false
	} else if err != nil {
		log.Error(err, "unable to verify, if dynatrace-bootstrapper-config is available, injection not possible")

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
