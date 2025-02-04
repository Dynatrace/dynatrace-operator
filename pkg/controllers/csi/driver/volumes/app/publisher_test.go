package app

import (
	"context"
	"os"
	"testing"
	"time"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestPublishVolume(t *testing.T) {
	ctx := context.Background()
	path := metadata.PathResolver{}

	t.Run("early return - FS problem during timeout check == skip mounting", func(t *testing.T) {
		fs := getFailFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("early return - retry limit reached", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		require.NoError(t, fs.Mkdir(path.AppMountForID(volumeCfg.VolumeID), os.ModePerm))

		pastTime := timeprovider.New()
		pastTime.Set(time.Now().Add(2 * volumeCfg.RetryTimeout))
		pub := Publisher{
			fs:      fs,
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
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("early return (with error) - binary is just a file", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		file, err := fs.Create(path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("early return (with error) - ruxit.conf not present", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		binaryDir := path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)
		require.NoError(t, fs.MkdirAll(binaryDir, os.ModePerm))

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)

		assert.Empty(t, mounter.MountPoints)
	})

	t.Run("happy path", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)

		// Binary present
		binaryDir := path.LatestAgentBinaryForDynaKube(volumeCfg.DynakubeName)
		require.NoError(t, fs.MkdirAll(binaryDir, os.ModePerm))

		// Config present
		confFile := path.AgentSharedRuxitAgentProcConf(volumeCfg.DynakubeName)
		conf := []byte("testing")
		require.NoError(t, fs.WriteFile(confFile, conf, os.ModePerm))

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Directories created correctly
		targetDirExists, _ := fs.IsDir(volumeCfg.TargetPath)
		assert.True(t, targetDirExists)

		varDir := path.AppMountVarDir(volumeCfg.VolumeID)
		varDirExits, _ := fs.IsDir(varDir)
		assert.True(t, varDirExits)

		mappedDir := path.AppMountMappedDir(volumeCfg.VolumeID)
		mappedDirExits, _ := fs.IsDir(mappedDir)
		assert.True(t, mappedDirExits)

		workDir := path.AppMountWorkDir(volumeCfg.VolumeID)
		workDirExits, _ := fs.IsDir(workDir)
		assert.True(t, workDirExits)

		// Config copied correctly, original untouched
		copiedConfFile := path.OverlayVarRuxitAgentProcConf(volumeCfg.VolumeID)
		copiedConf, err := fs.ReadFile(copiedConfFile)
		require.NoError(t, err)
		assert.Equal(t, conf, copiedConf)

		originalConf, err := fs.ReadFile(confFile)
		require.NoError(t, err)
		assert.Equal(t, conf, originalConf)

		// Mount happened
		// mounter.IsMountPoint can't be used as it uses os.Stat
		require.Len(t, mounter.MountPoints, 2)
		overlayMount := mounter.MountPoints[0]
		assert.Equal(t, "overlay", overlayMount.Device)
		assert.Equal(t, mappedDir, overlayMount.Path)
		require.Len(t, overlayMount.Opts, 3)
		assert.Contains(t, overlayMount.Opts[0], binaryDir) // lowerdir
		assert.Contains(t, overlayMount.Opts[1], varDir)    // upperdir
		assert.Contains(t, overlayMount.Opts[2], workDir)   // workdir

		bindMount := mounter.MountPoints[1]
		assert.Equal(t, "overlay", bindMount.Device) // this is set to "overlay" by the FakeMounter to mimic a linux FS
		assert.Equal(t, volumeCfg.TargetPath, bindMount.Path)
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
		RetryTimeout: time.Minute,
	}
}
