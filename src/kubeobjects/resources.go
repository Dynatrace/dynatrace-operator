package kubeobjects

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func NewResources(cpu, memory string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *NewQuantity(cpu),
		corev1.ResourceMemory: *NewQuantity(memory),
	}
}

func NewQuantity(serialized string) *resource.Quantity {
	return address.Of(resource.MustParse(serialized))
}
