package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestPublishVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("early return - FS problem during timeout check == skip mounting", func(t *testing.T) {
		problematicFolder := filepath.Join(t.TempDir(), "boom")
		require.NoError(t, os.MkdirAll(problematicFolder, 0444)) // r--r--r--, "readonly"

		t.Cleanup(func() {
			// needed, otherwise the `problematicFolder` wont be cleaned up after the test
			os.Chmod(problematicFolder, 0755)
		})

		path := metadata.PathResolver{RootDir: problematicFolder}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		pub := NewPublisher(mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("early return - retry limit reached", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		require.NoError(t, os.MkdirAll(path.AppMountForID(volumeCfg.VolumeID), os.ModePerm))

		pastTime := timeprovider.New()
		pastTime.Set(time.Now().Add(2 * volumeCfg.RetryTimeout))
		pub := Publisher{
			mounter: mounter,
			path:    path,
			time:    pastTime,
		}

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("early return (with error) - no binary present", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		pub := NewPublisher(mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("early return (with error) - binary is just a file", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		require.NoError(t, os.MkdirAll(filepath.Dir(path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)), os.ModePerm))
		file, err := os.Create(path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		pub := NewPublisher(mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("happy path", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		// Binary present
		binaryDir := path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)
		testBinary := path.AgentSharedBinaryDirForAgent("test")
		require.NoError(t, os.MkdirAll(filepath.Dir(binaryDir), os.ModePerm))
		require.NoError(t, os.MkdirAll(testBinary, os.ModePerm))
		require.NoError(t, os.Symlink(testBinary, binaryDir))

		pub := NewPublisher(mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Directories created correctly
		assert.DirExists(t, volumeCfg.TargetPath)

		varDir := path.AppMountVarDir(volumeCfg.VolumeID)
		assert.DirExists(t, varDir)

		mappedDir := path.AppMountMappedDir(volumeCfg.VolumeID)
		assert.DirExists(t, mappedDir)

		workDir := path.AppMountWorkDir(volumeCfg.VolumeID)
		assert.DirExists(t, workDir)

		isMountPoint, err := mounter.IsMountPoint(mappedDir)
		require.NoError(t, err)
		assert.True(t, isMountPoint)

		overlayMount := mounter.MountPoints[0]
		assert.Equal(t, "overlay", overlayMount.Device)
		assert.Contains(t, overlayMount.Path, mappedDir)
		require.Len(t, overlayMount.Opts, 3)
		assert.Contains(t, overlayMount.Opts[0], testBinary) // lowerdir
		assert.Contains(t, overlayMount.Opts[1], varDir)     // upperdir
		assert.Contains(t, overlayMount.Opts[2], workDir)    // workdir

		isMountPoint, err = mounter.IsMountPoint(volumeCfg.TargetPath)
		require.NoError(t, err)
		assert.True(t, isMountPoint)

		bindMount := mounter.MountPoints[1]
		if runtime.GOOS == "darwin" {
			assert.Equal(t, mappedDir, bindMount.Device)
		} else {
			// this is what linux does, and what we actually care about
			assert.Equal(t, "overlay", bindMount.Device)
		}
		assert.Equal(t, volumeCfg.TargetPath, bindMount.Path)
	})
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
		RetryTimeout: time.Minute,
	}
}
