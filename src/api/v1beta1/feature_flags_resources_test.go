package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ ResourceRequirementer = (*testResourceRequirementer)(nil)

type testResourceRequirementer struct {
	resources corev1.ResourceRequirements
}

func newTestResourceRequirementer() *testResourceRequirementer {
	return &testResourceRequirementer{
		resources: corev1.ResourceRequirements{
			Limits:   make(corev1.ResourceList),
			Requests: make(corev1.ResourceList),
		},
	}
}

func (testRequirementer *testResourceRequirementer) Limits(resourceName corev1.ResourceName) *resource.Quantity {
	if quantity, ok := testRequirementer.resources.Limits[resourceName]; ok {
		return &quantity
	}
	return nil
}

func (testRequirementer *testResourceRequirementer) Requests(resourceName corev1.ResourceName) *resource.Quantity {
	if quantity, ok := testRequirementer.resources.Requests[resourceName]; ok {
		return &quantity
	}
	return nil
}

func TestBuildResourceRequirements(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		testRequirementer := newTestResourceRequirementer()
		expectedRequests, expectedLimits := &testRequirementer.resources.Requests, &testRequirementer.resources.Limits

		testRequirementer.resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewScaledQuantity(500, resource.Milli),
			corev1.ResourceMemory: *resource.NewScaledQuantity(200, resource.Mega),
		}
		testRequirementer.resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewScaledQuantity(1000, resource.Milli),
		}

		requirements := BuildResourceRequirements(testRequirementer)

		assert.Equal(t, *expectedRequests.Cpu(), *requirements.Requests.Cpu())
		assert.Equal(t, *expectedRequests.Memory(), *requirements.Requests.Memory())
		assert.Equal(t, *expectedLimits.Cpu(), *requirements.Limits.Cpu())
		assert.Empty(t, requirements.Limits[corev1.ResourceMemory])
	})

	t.Run("empty resource requirements", func(t *testing.T) {
		testRequirementer := newTestResourceRequirementer()

		requirements := BuildResourceRequirements(testRequirementer)

		assert.Empty(t, requirements.Requests[corev1.ResourceCPU])
		assert.Empty(t, requirements.Requests[corev1.ResourceMemory])
		assert.Empty(t, requirements.Limits[corev1.ResourceCPU])
		assert.Empty(t, requirements.Limits[corev1.ResourceMemory])
	})
}
