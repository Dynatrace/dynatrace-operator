package webhook

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
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
	Mutate(ctx context.Context, request *MutationRequest) error

	// Reinvocation mutates the pod of the given ReinvocationRequest.
	// It only mutates the parts of the pod that haven't been mutated yet. (example: another webhook mutated the pod after our webhook was executed)
	Reinvoke(request *ReinvocationRequest) bool
}

// BaseRequest is the base request for all mutation requests
type BaseRequest struct {
	Pod       *corev1.Pod
	Namespace corev1.Namespace
	DynaKube  dynakube.DynaKube
}

func (req *BaseRequest) PodName() string {
	if req.Pod == nil {
		return ""
	}

	return pod.GetName(*req.Pod)
}

func (req *BaseRequest) NewContainers(isInjected func(corev1.Container) bool) (newContainers []*corev1.Container) {
	newContainers = []*corev1.Container{}

	for i := range req.Pod.Spec.Containers {
		container := &req.Pod.Spec.Containers[i]
		if IsContainerExcludedFromInjection(req.DynaKube.Annotations, req.Pod.Annotations, container.Name) {
			continue
		}

		if isInjected(*container) {
			continue
		}

		newContainers = append(newContainers, container)
	}

	return
}

// MutationRequest contains all the information needed to mutate a pod
// It is meant to be passed into each mutator, so that they can mutate the elements in the way they need to,
// and after passing it in to all the mutator the request will have the final state which can be used to mutate the pod.
type MutationRequest struct {
	*BaseRequest
	Context          context.Context
	InstallContainer *corev1.Container
}

// ReinvocationRequest contains all the information needed to reinvoke a pod
// It is meant to be passed into each mutator, so that they can mutate the elements in the way they need to,
// and after passing it in to all the mutator the request will have the final state which can be used to mutate the pod.
type ReinvocationRequest struct {
	*BaseRequest
}

func newBaseRequest(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) *BaseRequest {
	return &BaseRequest{
		Pod:       pod,
		DynaKube:  dk,
		Namespace: namespace,
	}
}

func NewMutationRequest(ctx context.Context, namespace corev1.Namespace, installContainer *corev1.Container, pod *corev1.Pod, dk dynakube.DynaKube) *MutationRequest {
	return &MutationRequest{
		BaseRequest:      newBaseRequest(pod, namespace, dk),
		Context:          ctx,
		InstallContainer: installContainer,
	}
}

func (request *MutationRequest) ToReinvocationRequest() *ReinvocationRequest {
	return &ReinvocationRequest{
		BaseRequest: request.BaseRequest,
	}
}
