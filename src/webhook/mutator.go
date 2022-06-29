package webhook

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type PodMutator interface {
	// Enabled returns true if the mutator needs to be executed for the given request.
	// This is used to filter out mutators that are not needed for the given request.
	Enabled(request *BaseRequest) bool

	// Injected returns true if the mutator has already injected into the pod of the given request.
	// This is used during reinvocation to prevent multiple injections.
	Injected(request *BaseRequest) bool

	// Mutate mutates the elements of the given MutationRequest, specifically the pod and installContainer.
	Mutate(request *MutationRequest) error

	// Reinvocation mutates the pod of the given ReinvocationRequest.
	// It only mutates the parts of the pod that haven't been mutated yet. (example: another webhook mutated the pod after our webhook was executed)
	Reinvoke(request *ReinvocationRequest) bool
}

// BaseRequest is the base request for all mutation requests
type BaseRequest struct {
	Pod      *corev1.Pod
	DynaKube dynatracev1beta1.DynaKube
}

// MutationRequest contains all the information needed to mutate a pod
// It is meant to be passed into each mutator, so that they can mutate the elements in the way they need to,
// and after passing it in to all the mutator the request will have the final state which can be used to mutate the pod.
type MutationRequest struct {
	*BaseRequest
	Context          context.Context
	Namespace        corev1.Namespace
	InstallContainer *corev1.Container
}

// ReinvocationRequest contains all the information needed to reinvoke a pod
// It is meant to be passed into each mutator, so that they can mutate the elements in the way they need to,
// and after passing it in to all the mutator the request will have the final state which can be used to mutate the pod.
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
