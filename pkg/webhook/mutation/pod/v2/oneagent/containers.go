package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	corev1 "k8s.io/api/core/v1"
)

const (
	isInjectedEnv = "DT_CM_INJECTED"
)

func Mutate(request *dtwebhook.MutationRequest) bool {
	installPath := oacommon.DefaultInstallPath // TODO: configure?
	mutateInitContainer(request, installPath)

	return mutateUserContainers(request.BaseRequest, installPath)
}

func Reinvoke(request *dtwebhook.BaseRequest) bool {
	installPath := oacommon.DefaultInstallPath // TODO: configure?

	return mutateUserContainers(request, installPath)
}

func containerIsInjected(container corev1.Container) bool {
	return env.IsIn(container.Env, isInjectedEnv)
}

func mutateUserContainers(request *dtwebhook.BaseRequest, installPath string) bool {
	newContainers := request.NewContainers(containerIsInjected)
	for i := range newContainers {
		container := newContainers[i]
		addOneAgentToContainer(request.DynaKube, container, installPath)
		setIsInjectedEnv(container)
	}

	return len(newContainers) > 0
}

func addOneAgentToContainer(dk dynakube.DynaKube, container *corev1.Container, installPath string) {
	log.Info("adding OneAgent to container", "name", container.Name)

	addVolumeMounts(container, installPath)
	oacommon.AddDeploymentMetadataEnv(container, dk)
	oacommon.AddPreloadEnv(container, installPath)
}

func setIsInjectedEnv(container *corev1.Container) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  isInjectedEnv,
			Value: "true",
		},
	)
}
