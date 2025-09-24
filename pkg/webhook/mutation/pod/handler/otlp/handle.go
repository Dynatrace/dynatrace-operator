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
	if h.envVarMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := h.envVarMutator.Mutate(mutationRequest)
		if err != nil {
			return err
		}
	}

	if h.resourceAttributeMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := h.resourceAttributeMutator.Mutate(mutationRequest)
		if err != nil {
			return err
		}
	}

	log.Debug("OTLP injection finished", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}
