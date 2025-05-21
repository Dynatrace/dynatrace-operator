package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	podv2 "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/oneagent"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	v1 dtwebhook.PodInjector
	v2 dtwebhook.PodInjector

	recorder events.EventRecorder
	decoder  admission.Decoder

	apiReader client.Reader

	webhookNamespace string
	deployedViaOLM   bool
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

	wh.recorder.Setup(mutationRequest)

	if podv2.IsEnabled(mutationRequest) {
		err := wh.v2.Handle(ctx, mutationRequest)
		if err != nil {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	} else {
		err := wh.v1.Handle(ctx, mutationRequest)
		if err != nil {
			return silentErrorResponse(mutationRequest.Pod, err)
		}
	}

	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}

	enabledOnPod := maputils.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)

	enabledOnContainers := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		enabledOnContainers = enabledOnContainers || !dtwebhook.IsContainerExcludedFromInjection(mutationRequest.DynaKube.Annotations, mutationRequest.Pod.Annotations, container.Name)
	}

	return enabledOnPod && enabledOnContainers
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

// ##############################################################################################################################

type Injector struct {
	recorder   events.EventRecorder
	kubeClient client.Client
	apiReader  client.Reader
	metaClient client.Client
}

func IsEnabled(mutationRequest *dtwebhook.MutationRequest) bool {
	ffEnabled := mutationRequest.DynaKube.FF().IsNodeImagePull()
	oaEnabled := oacommon.IsEnabled(mutationRequest.BaseRequest)

	defaultVolumeType := oacommon.EphemeralVolumeType
	if mutationRequest.DynaKube.OneAgent().IsCSIAvailable() {
		defaultVolumeType = oacommon.CSIVolumeType
	}

	correctVolumeType := maputils.GetField(mutationRequest.Pod.Annotations, oacommon.AnnotationVolumeType, defaultVolumeType) == oacommon.EphemeralVolumeType

	return ffEnabled && oaEnabled && correctVolumeType
}

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(kubeClient client.Client, apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder) *Injector {
	return &Injector{
		recorder:   recorder,
		kubeClient: kubeClient,
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
