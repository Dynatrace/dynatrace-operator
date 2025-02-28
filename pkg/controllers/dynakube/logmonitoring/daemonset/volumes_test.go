package daemonset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedMountLen     = 5
	expectedInitMountLen = 1
)

func TestGetVolumeMounts(t *testing.T) {
	tenantUUID := "test-uuid"

	t.Run("get volume mounts", func(t *testing.T) {
		mounts := getVolumeMounts(tenantUUID)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})
}

func TestGetVolumes(t *testing.T) {
	dkName := "test-dk"

	t.Run("get volumes", func(t *testing.T) {
		volumes := getVolumes(dkName)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			assert.NotEmpty(t, volume.VolumeSource)
		}
	})
}
