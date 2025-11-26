package k8sresource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func NewResourceList(cpu, memory string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *NewQuantity(cpu),
		corev1.ResourceMemory: *NewQuantity(memory),
	}
}

func NewQuantity(serialized string) *resource.Quantity {
	parsed := resource.MustParse(serialized)

	return &parsed
}
