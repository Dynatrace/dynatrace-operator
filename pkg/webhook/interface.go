package webhook

import "context"

type PodInjector interface {
	Handle(context.Context, *MutationRequest) error
}

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
