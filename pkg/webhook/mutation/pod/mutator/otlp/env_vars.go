package otlp

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
)

var (
	log = logd.Get().WithName("otlp-pod-mutation")
)

type EnvVarMutator struct{}

func (EnvVarMutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP env var injection is enabled")
	return false
}

func (EnvVarMutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP env vars have already been injected")
	return false
}

func (EnvVarMutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Debug("injecting OTLP env vars")
	return nil
}

func (EnvVarMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP env vars mutator")
	return false
}
