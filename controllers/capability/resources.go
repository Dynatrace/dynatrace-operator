package capability

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// Usage of SI-Prefix Mega instead of IEC-Prefix Mebi to make use of
	// scaling provided by resource.*. E.g., resource.Milli
	ResourceMemoryMinimum = "250M"
	ResourceCPUMinimum    = "150m"
	ResourceMemoryMaximum = "1G"
	ResourceCPUMaximum    = "300m"
)

func BuildResources(instance *dynatracev1alpha1.CapabilityProperties) corev1.ResourceRequirements {
	limits := buildResourceLimits(instance.Resources.Limits)
	requests := buildResourceRequests(instance.Resources.Requests, limits)

	return corev1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}

func buildResourceRequests(resourceList corev1.ResourceList, limits corev1.ResourceList) corev1.ResourceList {
	cpuMin := resource.MustParse(ResourceCPUMinimum)
	cpuRequest, hasCPURequest := resourceList[corev1.ResourceCPU]
	if !hasCPURequest {
		cpuRequest = cpuMin
	}

	memoryMin := resource.MustParse(ResourceMemoryMinimum)
	memoryRequest, hasMemoryRequest := resourceList[corev1.ResourceMemory]
	if !hasMemoryRequest {
		memoryRequest = memoryMin
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:    getMinResource(cpuRequest, limits[corev1.ResourceCPU]),
		corev1.ResourceMemory: getMinResource(memoryRequest, limits[corev1.ResourceMemory]),
	}
}

func buildResourceLimits(resourceList corev1.ResourceList) corev1.ResourceList {
	cpuLimit, hasCPULimit := resourceList[corev1.ResourceCPU]
	if !hasCPULimit {
		cpuLimit = resource.MustParse(ResourceCPUMaximum)
	}

	memoryLimit, hasMemoryLimit := resourceList[corev1.ResourceMemory]
	if !hasMemoryLimit {
		memoryLimit = resource.MustParse(ResourceMemoryMaximum)
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:    cpuLimit,
		corev1.ResourceMemory: memoryLimit,
	}
}

func getMinResource(a resource.Quantity, b resource.Quantity) resource.Quantity {
	if isASmallerThanB(a, b) {
		return a
	}
	return b
}

func isASmallerThanB(a resource.Quantity, b resource.Quantity) bool {
	return a.Cmp(b) < 0
}
