package annotations

import "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"

func SetDynatraceInjectedAnnotation(mutationRequest *mutator.MutationRequest, injectedAnnotation, reasonAnnotation string) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[injectedAnnotation] = "true"
	delete(mutationRequest.Pod.Annotations, reasonAnnotation)
}

func SetNotInjectedAnnotations(mutationRequest *mutator.MutationRequest, injectedAnnotation, reasonAnnotation, reasonValue string) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[injectedAnnotation] = "false"
	mutationRequest.Pod.Annotations[reasonAnnotation] = reasonValue
}
