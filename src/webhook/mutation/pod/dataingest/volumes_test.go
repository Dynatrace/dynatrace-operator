package dataingest_mutation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestSetupVolumes(t *testing.T) {
	t.Run("should add dataingest volumes", func(t *testing.T) {
		pod := &corev1.Pod{}

		setupVolumes(pod)

		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].Secret)
	})
}

func TestSetupVolumeMountsForUserContainer(t *testing.T) {
	t.Run("should add dataingest volume-mounts", func(t *testing.T) {
		container := &corev1.Container{}

		setupVolumeMountsForUserContainer(container)

		require.Len(t, container.VolumeMounts, 2)
	})
}
