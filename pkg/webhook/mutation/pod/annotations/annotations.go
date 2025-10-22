package annotations

import "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"

func SetDynatraceInjectedAnnotation(mutationRequest *mutator.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[mutator.AnnotationDynatraceInjected] = "true"
	delete(mutationRequest.Pod.Annotations, mutator.AnnotationDynatraceReason)
}

func SetNotInjectedAnnotations(mutationRequest *mutator.MutationRequest, reason string) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[mutator.AnnotationDynatraceInjected] = "false"
	mutationRequest.Pod.Annotations[mutator.AnnotationDynatraceReason] = reason
}
