package hostgroup

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	v1 "k8s.io/api/core/v1"
)

type Mutator struct{}

func NewMutator() *Mutator {
	return &Mutator{}
}

func (m *Mutator) Enabled(request *webhook.BaseRequest) bool {
	return needsHostGroup(request.DynaKube)
}

func needsHostGroup(dynakube v1beta1.DynaKube) bool {
	return dynakube.NeedAppInjection() && dynakube.HostGroup() != ""
}

func (m *Mutator) Injected(request *webhook.BaseRequest) bool {
	for _, initContainer := range request.Pod.Spec.InitContainers {
		if initContainer.Name == webhook.InstallContainerName && hasHostGroupEnvVar(initContainer) {
			return true
		}
	}

	return false
}

func hasHostGroupEnvVar(container v1.Container) bool {
	return kubeobjects.EnvVarIsIn(container.Env, EnvVarNameHostGroup)
}

func (m *Mutator) Mutate(request *webhook.MutationRequest) error {
	request.InstallContainer.Env = append(request.InstallContainer.Env, v1.EnvVar{
		Name:  EnvVarNameHostGroup,
		Value: request.DynaKube.HostGroup(),
	})

	return nil
}

func (m *Mutator) Reinvoke(request *webhook.ReinvocationRequest) bool {
	for _, initContainer := range request.Pod.Spec.InitContainers {
		if initContainer.Name == webhook.InstallContainerName {
			hostGroupEnvVar := kubeobjects.FindEnvVar(initContainer.Env, EnvVarNameHostGroup)
			if hostGroupEnvVar.Value != request.DynaKube.HostGroup() {
				hostGroupEnvVar.Value = request.DynaKube.HostGroup()
				return true
			}
		}
	}

	return false
}

var _ webhook.PodMutator = &Mutator{}
