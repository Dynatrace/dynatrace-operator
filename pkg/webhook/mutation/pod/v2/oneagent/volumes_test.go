package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddVolumeMounts(t *testing.T) {
	t.Run("should add oneagent volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		installPath := "test/path"

		addVolumeMounts(container, installPath)
		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, oneAgentCodeModulesVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, oneAgentCodeModulesConfigVolumeName, container.VolumeMounts[1].Name)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addInitVolumeMounts(container)
		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, oneAgentCodeModulesVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, oneAgentCodeModulesConfigVolumeName, container.VolumeMounts[1].Name)
	})
}

func TestAddOneAgentVolumes(t *testing.T) {
	t.Run("should add oneagent volumes", func(t *testing.T) {
		pod := &corev1.Pod{}

		addVolumes(pod)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.Equal(t, oneAgentCodeModulesVolumeName, pod.Spec.Volumes[0].Name)
		assert.Equal(t, oneAgentCodeModulesConfigVolumeName, pod.Spec.Volumes[1].Name)
	})
}
