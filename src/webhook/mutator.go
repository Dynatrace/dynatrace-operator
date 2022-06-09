package webhook

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type PodMutator interface {
	Enabled(request *BaseRequest) bool
	Injected(request *BaseRequest) bool
	Mutate(request *MutationRequest) error
	Reinvoke(request *ReinvocationRequest) bool
}

// BaseRequest is the base request for all mutation requests
type BaseRequest struct {
	Pod      *corev1.Pod
	DynaKube dynatracev1beta1.DynaKube
}

type MutationRequest struct {
	*BaseRequest
	Context          context.Context
	Namespace        corev1.Namespace
	InstallContainer *corev1.Container
}

type ReinvocationRequest struct {
	*BaseRequest
}

func newBaseRequest(pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) *BaseRequest {
	return &BaseRequest{
		Pod:      pod,
		DynaKube: dynakube,
	}
}

func NewMutationRequest(ctx context.Context, namespace corev1.Namespace, installContainer *corev1.Container, pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) *MutationRequest {
	return &MutationRequest{
		BaseRequest:      newBaseRequest(pod, dynakube),
		Context:          ctx,
		Namespace:        namespace,
		InstallContainer: installContainer,
	}
}

func NewReinvocationRequest(ctx context.Context, namespace corev1.Namespace, installContainer *corev1.Container, pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) *ReinvocationRequest {
	return &ReinvocationRequest{
		BaseRequest: newBaseRequest(pod, dynakube),
	}
}

func (request *MutationRequest) ToReinvocationRequest() *ReinvocationRequest {
	return &ReinvocationRequest{
		BaseRequest: request.BaseRequest,
	}
}
