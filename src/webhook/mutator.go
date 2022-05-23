package webhook

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type PodMutator interface {
	Enabled(pod *corev1.Pod) bool
	Mutate(request *MutationRequest) error
	Reinvoke(request *ReinvocationRequest) bool
}

type MutationRequest struct {
	Context       context.Context
	Pod           *corev1.Pod
	Namespace     *corev1.Namespace
	DynaKube      *dynatracev1beta1.DynaKube
	InitContainer *corev1.Container
}

type ReinvocationRequest struct {
	Pod      *corev1.Pod
	DynaKube *dynatracev1beta1.DynaKube
}

func (request *MutationRequest) ToReinvocationRequest() *ReinvocationRequest {
	return &ReinvocationRequest{
		Pod:      request.Pod,
		DynaKube: request.DynaKube,
	}
}

func FindInitContainer(initContainers []corev1.Container) *corev1.Container {
	for _, container := range initContainers {
		if container.Name == InstallContainerName {
			return &container
		}
	}
	return nil
}
