package otlp

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/annotations"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/secrets"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	kubeClient client.Client
	apiReader  client.Reader

	envVarMutator            dtwebhook.Mutator
	resourceAttributeMutator dtwebhook.Mutator
}

func New(
	kubeClient client.Client,
	apiReader client.Reader,
	envVarMutator dtwebhook.Mutator,
	resourceAttributeMutator dtwebhook.Mutator,
) *Handler {
	return &Handler{
		kubeClient:               kubeClient,
		apiReader:                apiReader,
		envVarMutator:            envVarMutator,
		resourceAttributeMutator: resourceAttributeMutator,
	}
}

func (h *Handler) Handle(mutationRequest *dtwebhook.MutationRequest) error {
	if !mutationRequest.DynaKube.OTLPExporterConfiguration().IsEnabled() {
		log.Debug("OTLP injection disabled", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

		return nil
	}

	if h.envVarMutator.IsEnabled(mutationRequest.BaseRequest) {
		if !h.isInputSecretPresent(
			mutationRequest,
			exporterconfig.GetSourceConfigSecretName(mutationRequest.DynaKube.Name),
			consts.OTLPExporterSecretName,
		) {
			log.Debug("required input secret not present, skipping OTLP injection", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

			return nil
		}

		if h.envVarMutator.IsInjected(mutationRequest.BaseRequest) {
			if h.envVarMutator.Reinvoke(mutationRequest.ToReinvocationRequest()) {
				log.Debug("reinvocation policy applied", "podName", mutationRequest.PodName())
			}
		} else {
			err := h.envVarMutator.Mutate(mutationRequest)
			if err != nil {
				return err
			}
		}
	}

	if h.resourceAttributeMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := h.resourceAttributeMutator.Mutate(mutationRequest)
		if err != nil {
			return err
		}
	}

	annotations.SetDynatraceInjectedAnnotation(
		mutationRequest,
		AnnotationOTLPInjected,
		AnnotationOTLPReason,
	)

	log.Debug("OTLP injection finished", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (h *Handler) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) bool {
	err := secrets.EnsureReplicated(mutationRequest, h.kubeClient, h.apiReader, sourceSecretName, targetSecretName, log)
	if k8serrors.IsNotFound(err) {
		annotations.SetNotInjectedAnnotations(
			mutationRequest,
			AnnotationOTLPInjected,
			AnnotationOTLPReason,
			NoOTLPExporterConfigSecretReason,
		)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		annotations.SetNotInjectedAnnotations(
			mutationRequest,
			dtwebhook.AnnotationDynatraceInjected,
			dtwebhook.AnnotationDynatraceReason,
			NoOTLPExporterConfigSecretReason,
		)

		return false
	}

	return true
}
