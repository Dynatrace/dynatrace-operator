package injection

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/annotations"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/secrets"
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

func New( //nolint:revive
	kubeClient client.Client,
	apiReader client.Reader,
	recorder events.EventRecorder,
	webhookPodImage string,
	isOpenShift bool,
	metaMutator,
	oaMutator dtwebhook.Mutator,
) *Handler {
	return &Handler{
		kubeClient:      kubeClient,
		apiReader:       apiReader,
		recorder:        recorder,
		webhookPodImage: webhookPodImage,
		isOpenShift:     isOpenShift,
		metaMutator:     metaMutator,
		oaMutator:       oaMutator,
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
			annotations.SetNotInjectedAnnotations(mutationRequest, NoMutationNeededReason)

			return nil
		}
	}

	annotations.SetDynatraceInjectedAnnotation(mutationRequest)

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
	err := secrets.EnsureReplicated(mutationRequest, h.kubeClient, h.apiReader, sourceSecretName, targetSecretName, log)
	if k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("unable to copy source of %s as it is not available, injection not possible", sourceSecretName), "pod", mutationRequest.PodName())

		annotations.SetNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		annotations.SetNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	return true
}
