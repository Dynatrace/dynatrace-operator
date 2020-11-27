package kubemon

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildResources(t *testing.T) {
	t.Run(`BuildResources with default values`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		resources := buildResources(instance)

		cpuLimit := resources.Limits[corev1.ResourceCPU]
		memoryLimit := resources.Limits[corev1.ResourceMemory]
		cpuRequest := resources.Requests[corev1.ResourceCPU]
		memoryRequest := resources.Requests[corev1.ResourceMemory]

		assert.True(t, resource.NewScaledQuantity(300, resource.Milli).Equal(cpuLimit))
		assert.True(t, resource.NewScaledQuantity(1, resource.Giga).Equal(memoryLimit))
		assert.True(t, resource.NewScaledQuantity(150, resource.Milli).Equal(cpuRequest))
		assert.True(t, resource.NewScaledQuantity(250, resource.Mega).Equal(memoryRequest))
	})
	t.Run(`BuildResources with custom resource requests and limits`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewScaledQuantity(500, resource.Milli),
							corev1.ResourceMemory: *resource.NewScaledQuantity(512, resource.Mega),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewScaledQuantity(180, resource.Milli),
							corev1.ResourceMemory: *resource.NewScaledQuantity(1024, resource.Mega)},
					}}}}
		resources := buildResources(instance)

		assert.NotNil(t, resources)

		cpuLimit := resources.Limits[corev1.ResourceCPU]
		memoryLimit := resources.Limits[corev1.ResourceMemory]
		cpuRequest := resources.Requests[corev1.ResourceCPU]
		memoryRequest := resources.Requests[corev1.ResourceMemory]

		assert.True(t, resource.NewScaledQuantity(300, resource.Milli).Equal(cpuLimit))
		assert.True(t, resource.NewScaledQuantity(512, resource.Mega).Equal(memoryLimit))
		assert.True(t, resource.NewScaledQuantity(180, resource.Milli).Equal(cpuRequest))
		assert.True(t, resource.NewScaledQuantity(512, resource.Mega).Equal(memoryRequest))
	})
}