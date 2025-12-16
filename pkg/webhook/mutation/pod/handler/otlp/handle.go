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

	// the execution of both the env var mutator and the resource attribute mutator
	// is controlled by the env var mutator's IsEnabled method
	// therefore, we only need to check it here
	if h.envVarMutator.IsEnabled(mutationRequest.BaseRequest) {
		if !h.isTokenSecretPresent(
			mutationRequest,
			exporterconfig.GetSourceConfigSecretName(mutationRequest.DynaKube.Name),
		) {
			log.Debug("required input secret not present, skipping OTLP injection", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

			return nil
		}

		if !h.isExporterActiveGateCertSecretPresent(mutationRequest,
			exporterconfig.GetSourceCertsSecretName(mutationRequest.DynaKube.Name),
		) {
			log.Debug("required ActiveGate cert secret not present, skipping OTLP injection", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

			return nil
		}

		if h.envVarMutator.IsInjected(mutationRequest.BaseRequest) {
			if h.envVarMutator.Reinvoke(mutationRequest.ToReinvocationRequest()) {
				log.Debug("OTLP exporter env var reinvocation policy applied", "podName", mutationRequest.PodName())
			}

			if h.resourceAttributeMutator.Reinvoke(mutationRequest.ToReinvocationRequest()) {
				log.Debug("OTLP resource attribute reinvocation policy applied", "podName", mutationRequest.PodName())
			}
		} else {
			if err := h.envVarMutator.Mutate(mutationRequest); err != nil {
				return err
			}

			if err := h.resourceAttributeMutator.Mutate(mutationRequest); err != nil {
				return err
			}
		}
	}

	annotations.SetInjected(
		mutationRequest,
		dtwebhook.AnnotationOTLPInjected,
		dtwebhook.AnnotationOTLPReason,
	)

	log.Debug("OTLP injection finished", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (h *Handler) isTokenSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName string) bool {
	err := secrets.EnsureReplicated(mutationRequest, h.kubeClient, h.apiReader, sourceSecretName, consts.OTLPExporterSecretName, log)
	if k8serrors.IsNotFound(err) {
		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationOTLPInjected,
			dtwebhook.AnnotationOTLPReason,
			NoOTLPExporterConfigSecretReason,
		)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationOTLPInjected,
			dtwebhook.AnnotationOTLPReason,
			NoOTLPExporterConfigSecretReason,
		)

		return false
	}

	return true
}

func (h *Handler) isExporterActiveGateCertSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName string) bool {
	if !mutationRequest.DynaKube.ActiveGate().HasCaCert() && mutationRequest.DynaKube.Spec.TrustedCAs == "" {
		// no ActiveGate, no certs needed
		return true
	}

	err := secrets.EnsureReplicated(mutationRequest, h.kubeClient, h.apiReader, sourceSecretName, consts.OTLPExporterCertsSecretName, log)
	if k8serrors.IsNotFound(err) {
		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationOTLPInjected,
			dtwebhook.AnnotationOTLPReason,
			NoOTLPExporterActiveGateCertSecretReason,
		)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		annotations.SetNotInjected(
			mutationRequest,
			dtwebhook.AnnotationOTLPInjected,
			dtwebhook.AnnotationOTLPReason,
			NoOTLPExporterActiveGateCertSecretReason,
		)

		return false
	}

	return true
}
