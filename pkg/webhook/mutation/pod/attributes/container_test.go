package attributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewContainerAttributes(t *testing.T) {
	t.Run("captures container name", func(t *testing.T) {
		c := corev1.Container{Name: "my-container"}
		attrs := NewContainerAttributes(c)
		assert.Equal(t, "my-container", attrs.ContainerName)
	})

	t.Run("empty container name", func(t *testing.T) {
		c := corev1.Container{}
		attrs := NewContainerAttributes(c)
		assert.Empty(t, attrs.ContainerName)
	})
}

func TestContainerAttributes_ToMap(t *testing.T) {
	t.Run("returns map with K8sContainerNameAttr key", func(t *testing.T) {
		attrs := &ContainerAttributes{ContainerName: "my-container"}
		m := attrs.ToMap()
		assert.Equal(t, map[string]string{K8sContainerNameAttr: "my-container"}, m)
	})

	t.Run("empty container name produces empty value", func(t *testing.T) {
		attrs := &ContainerAttributes{}
		m := attrs.ToMap()
		assert.Equal(t, map[string]string{K8sContainerNameAttr: ""}, m)
	})
}
