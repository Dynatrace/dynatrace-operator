package resourceattributes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
)

var (
	log = logd.Get().WithName("otlp-resource-attributes-pod-mutation")
)

type Mutator struct{}

func New() dtwebhook.Mutator {
	return &Mutator{}
}

func (Mutator) IsEnabled(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP resource attributes injection is enabled")

	return false
}

func (Mutator) IsInjected(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking of OTLP resource attributes have already been injected")

	return false
}

func (Mutator) Mutate(_ *dtwebhook.MutationRequest) error {
	log.Debug("injecting OTLP resource attributes")

	return nil
}

func (Mutator) Reinvoke(_ *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator")

	return false
}
