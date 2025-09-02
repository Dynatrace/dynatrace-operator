package host

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestPublishVolume(t *testing.T) {
	ctx := context.Background()
	pathResolver := metadata.PathResolver{}

	t.Run("happy path", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		pub := NewPublisher(fs, mounter, pathResolver)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		expectedHostDir := pathResolver.OsAgentDir(volumeCfg.DynakubeName)
		hostDirExists, _ := fs.IsDir(expectedHostDir)
		assert.True(t, hostDirExists)
		// mounter.IsMountPoint can't be used as it uses os.Stat
		require.Len(t, mounter.MountPoints, 1)
		hostMount := mounter.MountPoints[0]
		assert.Equal(t, expectedHostDir, hostMount.Device)
		assert.Equal(t, volumeCfg.TargetPath, hostMount.Path)
	})

	t.Run("sad path", func(t *testing.T) {
		fs := getFailFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		pub := NewPublisher(fs, mounter, pathResolver)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("handles dangling fs path", func(t *testing.T) {
		base := t.TempDir()
		pathResolver := metadata.PathResolver{RootDir: base}

		fs := afero.Afero{Fs: afero.NewOsFs()}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		// Create dir to be symlinked
		oldDir := pathResolver.OldOsAgentDir(volumeCfg.DynakubeName)
		require.NoError(t, os.MkdirAll(oldDir, os.ModePerm))

		// Create symlink to dir
		newDir := pathResolver.OsAgentDir(volumeCfg.DynakubeName)
		require.NoError(t, os.MkdirAll(filepath.Dir(newDir), os.ModePerm))
		require.NoError(t, os.Symlink(oldDir, newDir))

		// Remove dir where the symlink was pointing to -> create dangling symlink
		require.NoError(t, os.Remove(oldDir))

		_, err := fs.Stat(newDir)
		require.Error(t, err)

		pub := NewPublisher(fs, mounter, pathResolver)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		info, err := fs.Stat(newDir)
		require.NoError(t, err)
		require.NotNil(t, info)
	})
}

func getTestFs(t *testing.T) afero.Afero {
	t.Helper()

	return afero.Afero{Fs: afero.NewMemMapFs()}
}

func getFailFs(t *testing.T) afero.Afero {
	t.Helper()

	afero.NewReadOnlyFs(getTestFs(t))

	return afero.Afero{Fs: afero.NewReadOnlyFs(getTestFs(t))}
}

func getTestVolumeConfig(t *testing.T) csivolumes.VolumeConfig {
	t.Helper()

	return csivolumes.VolumeConfig{
		VolumeInfo: csivolumes.VolumeInfo{
			VolumeID:   "test-id",
			TargetPath: "test/path",
		},
		PodName:      "test-pod",
		Mode:         Mode,
		DynakubeName: "test-dk",
		RetryTimeout: time.Microsecond, // doesn't matter
	}
}

func Test_cleanupDanglingSymlink(t *testing.T) {
	t.Run("removes dangling symlink", func(t *testing.T) {
		base := t.TempDir()

		// Create dir to be symlinked
		missingDir := filepath.Join(base, "dir")
		require.NoError(t, os.MkdirAll(missingDir, os.ModePerm))

		// Create symlink to dir
		danglingLink := filepath.Join(base, "link")
		require.NoError(t, os.Symlink(missingDir, danglingLink))

		// Remove dir where the symlink was pointing to -> create dangling symlink
		require.NoError(t, os.Remove(missingDir))

		entries, err := os.ReadDir(base)
		require.NoError(t, err)
		require.Len(t, entries, 1) // check that only the link is there

		cleanupDanglingSymlink(danglingLink)

		entries, err = os.ReadDir(base)
		require.NoError(t, err)
		require.Empty(t, entries)
	})

	t.Run("leaves intact symlink", func(t *testing.T) {
		base := t.TempDir()

		// Create dir to be symlinked
		dir := filepath.Join(base, "dir")
		require.NoError(t, os.MkdirAll(dir, os.ModePerm))

		// Create symlink to dir
		link := filepath.Join(base, "link")
		require.NoError(t, os.Symlink(dir, link))

		entries, err := os.ReadDir(base)
		require.NoError(t, err)
		require.Len(t, entries, 2) // check that both the link and dir are there

		cleanupDanglingSymlink(link)

		entries, err = os.ReadDir(base)
		require.NoError(t, err)
		require.Len(t, entries, 2) // check that both the link and dir are there
	})
}
