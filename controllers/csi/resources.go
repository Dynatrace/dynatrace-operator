package dtcsi

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type containerResources struct {
	containerName        string
	resourcesMap         map[string]corev1.ResourceList
	defaultCpu           int64
	defaultMemoryRequest int64
	defaultMemoryLimit   int64
}

func (containerResource *containerResources) resourceRequirements() corev1.ResourceRequirements {
	resources := containerResource.resourcesMap[containerResource.containerName]

	cpu := getResource(containerResource.defaultCpu, resources, corev1.ResourceCPU, resource.Milli)
	memoryRequest := getResource(containerResource.defaultMemoryRequest, resources, corev1.ResourceMemory, resource.Mega)
	memoryLimit := getResource(containerResource.defaultMemoryLimit, resources, corev1.ResourceMemory, resource.Mega)

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: memoryRequest,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: memoryLimit,
		},
	}
}

func getResource(defaultValue int64, resources corev1.ResourceList, resourceType corev1.ResourceName, resourceQuantity resource.Scale) resource.Quantity {
	if resources != nil {
		resourceValue, ok := resources[resourceType]
		if ok && !resourceValue.IsZero() {
			return resourceValue
		}
	}
	return getQuantity(defaultValue, resourceQuantity)
}
