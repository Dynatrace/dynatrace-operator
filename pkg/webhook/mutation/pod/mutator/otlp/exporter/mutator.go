package exporter

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

type Mutator struct{}

func New() dtwebhook.Mutator {
	return &Mutator{}
}

func (Mutator) IsEnabled(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP env var injection is enabled")
	return false
}

func (Mutator) IsInjected(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP env vars have already been injected")
	return false
}

func (Mutator) Mutate(_ *dtwebhook.MutationRequest) error {
	log.Debug("injecting OTLP env vars")
	return nil
}

func (Mutator) Reinvoke(_ *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP env vars mutator")
	return false
}
