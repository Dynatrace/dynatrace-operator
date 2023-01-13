package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// +kubebuilder:object:generate=false
type ResourceRequirementer interface {
	Requests(corev1.ResourceName) *resource.Quantity
	Limits(corev1.ResourceName) *resource.Quantity
}

func ResourceNames() []corev1.ResourceName {
	return []corev1.ResourceName{
		corev1.ResourceCPU, corev1.ResourceMemory,
	}
}

func BuildResourceRequirements(resourceRequirementer ResourceRequirementer) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Limits:   make(corev1.ResourceList),
		Requests: make(corev1.ResourceList),
	}

	for _, resourceName := range ResourceNames() {
		if quantity := resourceRequirementer.Limits(resourceName); quantity != nil {
			requirements.Limits[resourceName] = *quantity
		}
		if quantity := resourceRequirementer.Requests(resourceName); quantity != nil {
			requirements.Requests[resourceName] = *quantity
		}
	}

	return requirements
}
