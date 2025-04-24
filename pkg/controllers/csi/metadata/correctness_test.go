package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestGetRelevantOverlayMounts(t *testing.T) {
	t.Run("get only relevant mounts", func(t *testing.T) {
		baseFolder := "/test/folder"
		expectedPath := baseFolder + "/some/sub/folder"
		expectedLowerDir := "/data/codemodules/cXVheS5pby9keW5hdHJhY2UvZHluYXRyYWNlLWJvb3RzdHJhcHBlcjpzbmFwc2hvdA=="
		expectedUpperDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/var"
		expectedWorkDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/work"

		relevantMountPoint := mount.MountPoint{
			Device: "overlay",
			Path:   expectedPath,
			Type:   "overlay",
			Opts: []string{
				"lowerdir=" + expectedLowerDir,
				"upperdir=" + expectedUpperDir,
				"workdir=" + expectedWorkDir,
			},
		}

		mounter := mount.NewFakeMounter([]mount.MountPoint{
			relevantMountPoint,
			{
				Device: "not-relevant-mount-type",
			},
			{
				Device: "overlay",
				Path:   "not-relevant-overlay-mount",
				Type:   "overlay",
			},
		})

		mounts, err := GetRelevantOverlayMounts(mounter, baseFolder)
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Len(t, mounts, 1)
		assert.Equal(t, expectedPath, mounts[0].Path)
		assert.Equal(t, expectedLowerDir, mounts[0].LowerDir)
		assert.Equal(t, expectedUpperDir, mounts[0].UpperDir)
		assert.Equal(t, expectedWorkDir, mounts[0].WorkDir)
	})

	t.Run("works with no mount points", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		mounts, err := GetRelevantOverlayMounts(mounter, "")
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Empty(t, mounts)
	})

	t.Run("ignores irrelevant mounts", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{
				Device: "not-relevant-mount-type",
			},
			{
				Device: "overlay",
				Path:   "not-relevant-overlay-mount",
				Type:   "overlay",
			},
		})
		mounts, err := GetRelevantOverlayMounts(mounter, "/test")
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Empty(t, mounts)
	})
}

func TestMigrateAppMounts(t *testing.T) {
	// Unfortunately, this is not unit-testable.
	// Its output would be a bunch of symlinks,
	// however the afero.MemMapFs does not support symlinking.
	t.SkipNow()
}

func TestMigrateHostMounts(t *testing.T) {
	// Unfortunately, this is not unit-testable.
	// Its output would be a bunch of symlinks,
	// however the afero.MemMapFs does not support symlinking.
	t.SkipNow()
}
