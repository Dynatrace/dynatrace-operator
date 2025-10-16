package otlp

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
)

type Handler struct {
	envVarMutator            dtwebhook.Mutator
	resourceAttributeMutator dtwebhook.Mutator
}

func New(
	envVarMutator dtwebhook.Mutator,
	resourceAttributeMutator dtwebhook.Mutator,
) *Handler {
	return &Handler{
		envVarMutator:            envVarMutator,
		resourceAttributeMutator: resourceAttributeMutator,
	}
}

func (h *Handler) Handle(mutationRequest *dtwebhook.MutationRequest) error {
	if !mutationRequest.DynaKube.OTLPExporterConfiguration().IsEnabled() {
		return nil
	}

	if h.envVarMutator.IsEnabled(mutationRequest.BaseRequest) {
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

	log.Debug("OTLP injection finished", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}
