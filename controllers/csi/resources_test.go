package dtcsi

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

const (
	testContainerName        = "test-container"
	testDefaultCPU           = 123
	testDefaultMemoryRequest = 234
	testDefaultMemoryLimit   = 345

	testDefaultCPUQuantity           = "123m"
	testDefaultMemoryRequestQuantity = "234M"
	testDefaultMemoryLimitQuantity   = "345M"
)

func TestResourceRequirements(t *testing.T) {
	resourceRequirements := (&containerResources{
		containerName:        testContainerName,
		resourcesMap:         nil,
		defaultCpu:           testDefaultCPU,
		defaultMemoryRequest: testDefaultMemoryRequest,
		defaultMemoryLimit:   testDefaultMemoryLimit,
	}).resourceRequirements()

	assert.Contains(t, resourceRequirements.Requests, corev1.ResourceCPU)
	assert.Contains(t, resourceRequirements.Requests, corev1.ResourceMemory)
	assert.Contains(t, resourceRequirements.Limits, corev1.ResourceCPU)
	assert.Contains(t, resourceRequirements.Limits, corev1.ResourceMemory)

	assert.True(t, resourceRequirements.Requests[corev1.ResourceCPU].Equal(resource.MustParse(testDefaultCPUQuantity)))
	assert.True(t, resourceRequirements.Requests[corev1.ResourceMemory].Equal(resource.MustParse(testDefaultMemoryRequestQuantity)))
	assert.True(t, resourceRequirements.Limits[corev1.ResourceCPU].Equal(resource.MustParse(testDefaultCPUQuantity)))
	assert.True(t, resourceRequirements.Limits[corev1.ResourceMemory].Equal(resource.MustParse(testDefaultMemoryLimitQuantity)))
}
