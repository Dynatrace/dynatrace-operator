package otlp

import (
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
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
	if !shouldInject(mutationRequest) {
		log.Debug("OTLP injection disabled", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)
		return nil
	}

	log.Debug("OTLP injection enabled", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

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

func shouldInject(request *dtwebhook.MutationRequest) bool {
	// first, check if otlp injection is enabled explicitly on pod
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationOTLPInjectionEnabled, false)

	if !enabledOnPod {
		// if not enabled explicitly, check general injection setting via 'dynatrace.com/inject' annotation
		enabledOnPod = maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, request.DynaKube.FF().IsAutomaticInjection())
	}

	// TODO also check for namespaceSelector of OTLP config when CRD has been updated
	namespaceEnabled := true

	return enabledOnPod && namespaceEnabled
}
