package capability

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultCpuLimit      = resource.NewScaledQuantity(300, resource.Milli)
	defaultMemoryLimit   = resource.NewScaledQuantity(1, resource.Giga)
	defaultCpuRequest    = resource.NewScaledQuantity(150, resource.Milli)
	defaultMemoryRequest = resource.NewScaledQuantity(250, resource.Mega)
)

func TestBuildResources_Defaults(t *testing.T) {
	resources := BuildResources(&v1alpha1.CapabilityProperties{})

	assert.NotNil(t, resources)
	assert.True(t, defaultCpuLimit.Equal(resources.Limits[corev1.ResourceCPU]))
	assert.True(t, defaultMemoryLimit.Equal(resources.Limits[corev1.ResourceMemory]))
	assert.True(t, defaultCpuRequest.Equal(resources.Requests[corev1.ResourceCPU]))
	assert.True(t, defaultMemoryRequest.Equal(resources.Requests[corev1.ResourceMemory]))
}

func TestBuildResources_HigherThenDefaultValues(t *testing.T) {
	quantityCpuLimit := resource.NewScaledQuantity(5000, resource.Milli)
	quantityMemoryLimit := resource.NewScaledQuantity(6000, resource.Mega)
	quantityCpuRequest := resource.NewScaledQuantity(4000, resource.Milli)
	quantityMemoryRequest := resource.NewScaledQuantity(3000, resource.Mega)

	resources := BuildResources(&v1alpha1.CapabilityProperties{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuLimit,
				corev1.ResourceMemory: *quantityMemoryLimit,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuRequest,
				corev1.ResourceMemory: *quantityMemoryRequest,
			},
		},
	})

	assert.NotNil(t, resources)
	assert.True(t, quantityCpuLimit.Equal(resources.Limits[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryLimit.Equal(resources.Limits[corev1.ResourceMemory]))
	assert.True(t, quantityCpuRequest.Equal(resources.Requests[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryRequest.Equal(resources.Requests[corev1.ResourceMemory]))
}

func TestBuildResources_LowerThenDefaultValues(t *testing.T) {
	quantityCpuLimit := resource.NewScaledQuantity(50, resource.Milli)
	quantityMemoryLimit := resource.NewScaledQuantity(60, resource.Mega)
	quantityCpuRequest := resource.NewScaledQuantity(40, resource.Milli)
	quantityMemoryRequest := resource.NewScaledQuantity(30, resource.Mega)

	resources := BuildResources(&v1alpha1.CapabilityProperties{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuLimit,
				corev1.ResourceMemory: *quantityMemoryLimit,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuRequest,
				corev1.ResourceMemory: *quantityMemoryRequest,
			},
		},
	})

	assert.NotNil(t, resources)
	assert.True(t, quantityCpuLimit.Equal(resources.Limits[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryLimit.Equal(resources.Limits[corev1.ResourceMemory]))
	assert.True(t, quantityCpuRequest.Equal(resources.Requests[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryRequest.Equal(resources.Requests[corev1.ResourceMemory]))
}

func TestBuildResources_HigherRequestsThenLimits(t *testing.T) {
	quantityCpuLimit := resource.NewScaledQuantity(50, resource.Milli)
	quantityMemoryLimit := resource.NewScaledQuantity(60, resource.Mega)
	quantityCpuRequest := resource.NewScaledQuantity(400, resource.Milli)
	quantityMemoryRequest := resource.NewScaledQuantity(300, resource.Mega)

	resources := BuildResources(&v1alpha1.CapabilityProperties{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuLimit,
				corev1.ResourceMemory: *quantityMemoryLimit,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *quantityCpuRequest,
				corev1.ResourceMemory: *quantityMemoryRequest,
			},
		},
	})

	assert.NotNil(t, resources)
	assert.True(t, quantityCpuLimit.Equal(resources.Limits[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryLimit.Equal(resources.Limits[corev1.ResourceMemory]))
	assert.True(t, quantityCpuLimit.Equal(resources.Requests[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryLimit.Equal(resources.Requests[corev1.ResourceMemory]))
}
