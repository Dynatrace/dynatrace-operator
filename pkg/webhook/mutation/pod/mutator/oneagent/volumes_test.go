package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
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
		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, installPath, container.VolumeMounts[0].MountPath)
		assert.True(t, container.VolumeMounts[0].ReadOnly)

		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[1].Name)
		assert.Equal(t, ldPreloadPath, container.VolumeMounts[1].MountPath)
		assert.Equal(t, ldPreloadSubPath, container.VolumeMounts[1].SubPath)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		readonly := false

		addInitBinMount(container, readonly)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, consts.AgentInitBinDirMount, container.VolumeMounts[0].MountPath)
		assert.Equal(t, readonly, container.VolumeMounts[0].ReadOnly)
	})

	t.Run("should add readonly init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		readonly := true

		addInitBinMount(container, readonly)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, consts.AgentInitBinDirMount, container.VolumeMounts[0].MountPath)
		assert.Equal(t, readonly, container.VolumeMounts[0].ReadOnly)
	})
}
