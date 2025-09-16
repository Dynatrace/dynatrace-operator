package otlp

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
)

type ResourceAttributesMutator struct{}

func (ResourceAttributesMutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP resource attributes injection is enabled")
	return false
}

func (ResourceAttributesMutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP resource attributes have already been injected")
	return false
}

func (ResourceAttributesMutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Debug("injecting OTLP resource attributes")
	return nil
}

func (ResourceAttributesMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator")
	return false
}
