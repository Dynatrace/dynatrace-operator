package oneagent

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddVolumeMounts(t *testing.T) {
	t.Run("should add oneagent volume mounts", func(t *testing.T) {
		container := &corev1.Container{
			Name: "test-container",
		}
		installPath := "test/path"

		addVolumeMounts(container, installPath)
		require.Len(t, container.VolumeMounts, 3)
		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, installPath, container.VolumeMounts[0].MountPath)
		assert.Equal(t, binSubPath, container.VolumeMounts[0].SubPath)

		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[1].Name)
		assert.Equal(t, ldPreloadPath, container.VolumeMounts[1].MountPath)
		assert.Equal(t, filepath.Join(volumes.InitConfigSubPath, ldPreloadSubPath), container.VolumeMounts[1].SubPath)

		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[2].Name)
		assert.Equal(t, filepath.Join(volumes.InitConfigSubPath, container.Name), container.VolumeMounts[2].SubPath)
		assert.Equal(t, volumes.ConfigMountPath, container.VolumeMounts[2].MountPath)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addInitVolumeMounts(container)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, binInitMountPath, container.VolumeMounts[0].MountPath)
		assert.Equal(t, binSubPath, container.VolumeMounts[0].SubPath)
	})
}
