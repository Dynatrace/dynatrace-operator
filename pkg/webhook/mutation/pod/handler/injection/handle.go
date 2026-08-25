// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package injection

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/annotations"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/secrets"
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
	ctx, log := logd.NewFromContext(mutationRequest.Context, "injection")
	mutationRequest.Context = ctx

	if !mutationRequest.DynaKube.OneAgent().IsAppInjectionNeeded() && !mutationRequest.DynaKube.MetadataEnrichment().IsEnabled() {
		log.Debug("injection disabled", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

		return nil
	}

	oaEnabled := h.oaMutator.IsEnabled(ctx, mutationRequest.BaseRequest)

	if installContainer := k8scontainer.FindInitInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName); installContainer != nil {
		h.handlePodReinvocation(mutationRequest, installContainer, oaEnabled)

		return nil
	}

	needsConfig := oaEnabled || h.metaMutator.IsEnabled(ctx, mutationRequest.BaseRequest)

	if needsConfig && !h.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceConfigSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitSecretName) {
		return nil
	}

	if oaEnabled && (mutationRequest.DynaKube.IsAGCertificateNeeded() || mutationRequest.DynaKube.Spec.TrustedCAs != "") {
		if !h.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceCertsSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitCertsSecretName) {
			return nil
		}
	}

	mutated, err := h.handlePodMutation(mutationRequest, needsConfig, oaEnabled)
	if err != nil {
		return err
	}

	if !mutated {
		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationDynatraceInjected,
			dtwebhook.AnnotationDynatraceReason,
			NoMutationNeededReason,
		)

		return nil
	}

	annotations.SetInjected(
		mutationRequest,
		dtwebhook.AnnotationDynatraceInjected,
		dtwebhook.AnnotationDynatraceReason,
	)

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (h *Handler) handlePodMutation(mutationRequest *dtwebhook.MutationRequest, needsConfig, oaEnabled bool) (bool, error) {
	mutationRequest.InstallContainer = h.createInitContainerBase(mutationRequest.Context, mutationRequest.Pod, mutationRequest.DynaKube)

	var mutated bool

	if needsConfig {
		err := h.metaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if oaEnabled {
		err := h.oaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if mutated {
		if err := addInitContainerToPod(mutationRequest.Context, mutationRequest.Pod, mutationRequest.InstallContainer); err != nil {
			return false, err
		}

		events.SendPodInjectEvent(h.recorder, &mutationRequest.DynaKube, mutationRequest.Pod)
	}

	return mutated, nil
}

func (h *Handler) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest, installContainer *corev1.Container, oaEnabled bool) {
	log := logd.FromContext(mutationRequest.Context)
	log.Debug("Dynatrace init-container already present, skipping mutation, doing reinvocation", "containerName", dtwebhook.InstallContainerName)

	updated, err := metadata.AddContainerAttributes(mutationRequest.BaseRequest, installContainer)
	if err != nil {
		log.Error(err, "failed to update container-attributes on the init container during reinvoke")

		return
	}

	if (oaEnabled && h.oaMutator.Reinvoke(mutationRequest.Context, mutationRequest.ToReinvocationRequest())) || updated {
		log.Info("reinvocation policy applied", "podName", mutationRequest.PodName())
		events.SendPodUpdateEvent(h.recorder, &mutationRequest.DynaKube, mutationRequest.Pod)

		return
	}

	log.Info("no change, all containers already injected", "podName", mutationRequest.PodName())
}

func (h *Handler) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) bool {
	log := logd.FromContext(mutationRequest.Context)

	err := secrets.EnsureReplicated(mutationRequest, h.kubeClient, h.apiReader, sourceSecretName, targetSecretName, log)
	if k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("unable to copy source of %s as it is not available, injection not possible", sourceSecretName), "pod", mutationRequest.PodName())

		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationDynatraceInjected,
			dtwebhook.AnnotationDynatraceReason,
			NoBootstrapperConfigReason,
		)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationDynatraceInjected,
			dtwebhook.AnnotationDynatraceReason,
			NoBootstrapperConfigReason,
		)

		return false
	}

	return true
}
