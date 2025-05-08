package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedMountLen  = 2
	expectedVolumeLen = 2
)

func TestGetMounts(t *testing.T) {
	t.Run("get volume mounts", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})

	t.Run("get cert mount", func(t *testing.T) {
		dk := getDynaKubeWithCerts(t)
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen+1)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})

	t.Run("get cert mount with automatic AG cert", func(t *testing.T) {
		dk := getDynaKubeWithAutomaticCerts(t)
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen+1)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})
}

func TestGetVolumes(t *testing.T) {
	t.Run("get volumes", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			assert.NotEmpty(t, volume.VolumeSource)
		}
	})

	t.Run("add cert volume", func(t *testing.T) {
		dk := getDynaKubeWithCerts(t)
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen+1)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			require.NotEmpty(t, volume.VolumeSource)

			if volume.Name == certVolumeName {
				assert.NotEmpty(t, volume.VolumeSource.Secret.SecretName)
			}
		}
	})

	t.Run("add cert volume with automatic AG cert", func(t *testing.T) {
		dk := getDynaKubeWithAutomaticCerts(t)
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen+1)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			require.NotEmpty(t, volume.VolumeSource)

			if volume.Name == certVolumeName {
				assert.NotEmpty(t, volume.VolumeSource.Secret.SecretName)
			}
		}
	})
}
