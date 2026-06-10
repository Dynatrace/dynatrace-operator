package metadataenrichment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetInitResources(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		m := &MetadataEnrichment{Spec: &Spec{}}
		assert.Nil(t, m.GetInitResources())
	})

	t.Run("returns configured resources when set", func(t *testing.T) {
		resources := &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}

		m := &MetadataEnrichment{Spec: &Spec{InitResources: resources}}
		assert.Equal(t, resources, m.GetInitResources())
	})
}
