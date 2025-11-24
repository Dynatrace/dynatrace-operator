package daemonset

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedMountLen     = 5
	expectedInitMountLen = 1
)

func TestGetVolumeMounts(t *testing.T) {
	t.Run("get volume mounts", func(t *testing.T) {
		mounts := getVolumeMounts()

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
			if mount.Name == dtLibVolumeName {
				assert.Empty(t, mount.SubPath)
			}
		}
	})
}

func TestGetVolumes(t *testing.T) {
	dkName := "test-dk"
	tenantUUID := "test-uuid"

	t.Run("get volumes", func(t *testing.T) {
		volumes := getVolumes(dkName, tenantUUID)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			require.NotEmpty(t, volume.VolumeSource)
			if volume.Name == dtLibVolumeName {
				assert.Equal(t, fmt.Sprintf(dtLibVolumeHostSubPathTemplate, tenantUUID), filepath.Base(volume.HostPath.Path))
			}
		}
	})
}
