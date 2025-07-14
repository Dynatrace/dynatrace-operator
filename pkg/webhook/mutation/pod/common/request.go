package common

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	corev1 "k8s.io/api/core/v1"
)

func NewMutationRequest(ctx context.Context, namespace corev1.Namespace, installContainer *corev1.Container, pod *corev1.Pod, dk dynakube.DynaKube) *MutationRequest {
	return &MutationRequest{
		BaseRequest:      newBaseRequest(pod, namespace, dk),
		Context:          ctx,
		InstallContainer: installContainer,
	}
}

// MutationRequest contains all the information needed to mutate a pod
// It is meant to be passed into each mutator, so that they can mutate the elements in the way they need to,
// and after passing it in to all the mutator the request will have the final state which can be used to mutate the pod.
type MutationRequest struct {
	*BaseRequest
	Context          context.Context
	InstallContainer *corev1.Container
}

func (request *MutationRequest) ToReinvocationRequest() *ReinvocationRequest {
	return &ReinvocationRequest{
		BaseRequest: request.BaseRequest,
	}
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
