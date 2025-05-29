package v2

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/oneagent"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Injector struct {
	recorder    events.EventRecorder
	kubeClient  client.Client
	apiReader   client.Reader
	metaClient  client.Client
	isOpenShift bool
}

func IsEnabled(mutationRequest *dtwebhook.MutationRequest) bool {
	ffEnabled := mutationRequest.DynaKube.FeatureNodeImagePull()
	oaEnabled := oacommon.IsEnabled(mutationRequest.BaseRequest)

	defaultVolumeType := oacommon.EphemeralVolumeType
	if mutationRequest.DynaKube.OneAgent().IsCSIAvailable() {
		defaultVolumeType = oacommon.CSIVolumeType
	}

	correctVolumeType := maputils.GetField(mutationRequest.Pod.Annotations, oacommon.AnnotationVolumeType, defaultVolumeType) == oacommon.EphemeralVolumeType

	return ffEnabled && oaEnabled && correctVolumeType
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(kubeClient client.Client, apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder, isOpenShift bool) *Injector {
	return &Injector{
		recorder:    recorder,
		kubeClient:  kubeClient,
		apiReader:   apiReader,
		metaClient:  metaClient,
		isOpenShift: isOpenShift,
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
	mutationRequest.InstallContainer = createInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube, wh.isOpenShift)

	err := addContainerAttributes(mutationRequest)
	if err != nil {
		return err
	}

	updated := oamutation.Mutate(mutationRequest)
	if !updated {
		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoMutationNeededReason)

		return nil
	}

	err = wh.addPodAttributes(mutationRequest)
	if err != nil {
		log.Info("failed to add pod attributes to init-container")

		return err
	}

	oacommon.SetInjectedAnnotation(mutationRequest.Pod)

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	wh.recorder.SendPodInjectEvent()

	return nil
}

func (wh *Injector) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	mutationRequest.InstallContainer = container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)

	err := addContainerAttributes(mutationRequest)
	if err != nil {
		log.Error(err, "error during reinvocation for updating the init-container, failed to update container-attributes on the init container")

		return false
	}

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
	err := wh.replicateInputSecret(mutationRequest)

	if k8serrors.IsNotFound(err) {
		log.Info("unable to copy source of dynatrace-bootstrapper-config as it is not available, injection not possible", "pod", mutationRequest.PodName())

		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoBootstrapperConfigReason)

		return false
	}

	if err != nil {
		log.Error(err, "unable to verify, if dynatrace-bootstrapper-config is available, injection not possible")

		oacommon.SetNotInjectedAnnotations(mutationRequest.Pod, NoBootstrapperConfigReason)

		return false
	}

	return true
}

func (wh *Injector) replicateInputSecret(mutationRequest *dtwebhook.MutationRequest) error {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: mutationRequest.Namespace.Name}
	err := wh.apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret)

	if k8serrors.IsNotFound(err) {
		log.Info("dynatrace-bootstrapper-config is not available, trying to replicate", "pod", mutationRequest.PodName())

		return bootstrapperconfig.Replicate(mutationRequest.Context, mutationRequest.DynaKube, secret.Query(wh.kubeClient, wh.apiReader, log), mutationRequest.Namespace.Name)
	}

	return nil
}

func setDynatraceInjectedAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
	delete(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceReason)
}
