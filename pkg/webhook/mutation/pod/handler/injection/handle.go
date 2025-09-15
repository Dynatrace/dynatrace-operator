package injection

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	recorder    events.EventRecorder
	metaMutator dtwebhook.Mutator
	oaMutator   dtwebhook.Mutator

	kubeClient client.Client
	apiReader  client.Reader

	webhookPodImage string
	isOpenShift     bool
}

func New(
	kubeClient client.Client,
	apiReader client.Reader,
	recorder events.EventRecorder,
	webhookPodImage string,
	isOpenShift bool,
	metaMutator,
	oaMutator dtwebhook.Mutator,
) *Handler {
	return &Handler{
		recorder:        recorder,
		metaMutator:     metaMutator,
		oaMutator:       oaMutator,
		kubeClient:      kubeClient,
		apiReader:       apiReader,
		webhookPodImage: webhookPodImage,
		isOpenShift:     isOpenShift,
	}
}

func (h *Handler) Handle(mutationRequest *dtwebhook.MutationRequest) error {
	h.recorder.Setup(mutationRequest)

	if !h.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceConfigSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitSecretName) {
		return nil
	}

	if mutationRequest.DynaKube.IsAGCertificateNeeded() || mutationRequest.DynaKube.Spec.TrustedCAs != "" {
		if !h.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceCertsSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitCertsSecretName) {
			return nil
		}
	}

	if h.isInjected(mutationRequest) {
		if h.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName())
			h.recorder.SendPodUpdateEvent()

			return nil
		}

		log.Info("no change, all containers already injected", "podName", mutationRequest.PodName())

		return nil
	} else {
		mutated, err := h.handlePodMutation(mutationRequest)
		if err != nil {
			return err
		}

		if !mutated {
			setNotInjectedAnnotations(mutationRequest, NoMutationNeededReason)

			return nil
		}
	}

	setDynatraceInjectedAnnotation(mutationRequest)

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (h *Handler) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	installContainer := container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)
	if installContainer != nil {
		log.Info("Dynatrace init-container already present, skipping mutation, doing reinvocation", "containerName", dtwebhook.InstallContainerName)

		return true
	}

	return false
}

func (h *Handler) handlePodMutation(mutationRequest *dtwebhook.MutationRequest) (bool, error) {
	mutationRequest.InstallContainer = h.createInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)

	var mutated bool

	if h.oaMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := h.oaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if h.metaMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := h.metaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if mutated {
		_, err := addContainerAttributes(mutationRequest)
		if err != nil {
			return false, err
		}

		err = addPodAttributes(mutationRequest)
		if err != nil {
			log.Info("failed to add pod attributes to init-container")

			return false, err
		}

		addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
		h.recorder.SendPodInjectEvent()
	}

	return mutated, nil
}

func (h *Handler) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	mutationRequest.InstallContainer = container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)

	// metadata enrichment does not need to be reinvoked, addContainerAttributes() does what is needed
	hasNewContainers, err := addContainerAttributes(mutationRequest)
	if err != nil {
		log.Error(err, "error during reinvocation for updating the init-container, failed to update container-attributes on the init container")

		return false
	}

	var oaUpdated bool
	if h.oaMutator.IsEnabled(mutationRequest.BaseRequest) {
		oaUpdated = h.oaMutator.Reinvoke(mutationRequest.ToReinvocationRequest())
	}

	return hasNewContainers || oaUpdated
}

func (h *Handler) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) bool {
	err := h.replicateSecret(mutationRequest, sourceSecretName, targetSecretName)
	if k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("unable to copy source of %s as it is not available, injection not possible", sourceSecretName), "pod", mutationRequest.PodName())

		setNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		setNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	return true
}

func (h *Handler) replicateSecret(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) error {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: targetSecretName, Namespace: mutationRequest.Namespace.Name}

	err := h.apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret)
	if k8serrors.IsNotFound(err) {
		log.Info(targetSecretName+" is not available, trying to replicate", "pod", mutationRequest.PodName())

		return bootstrapperconfig.Replicate(mutationRequest.Context, mutationRequest.DynaKube, secret.Query(h.kubeClient, h.apiReader, log), sourceSecretName, targetSecretName, mutationRequest.Namespace.Name)
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

func setNotInjectedAnnotations(mutationRequest *dtwebhook.MutationRequest, reason string) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "false"
	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceReason] = reason
}
