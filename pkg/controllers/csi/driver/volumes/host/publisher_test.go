package host

import (
	"context"
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
	path := metadata.PathResolver{}

	t.Run("happy path", func(t *testing.T) {
		fs := getTestFs(t)
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		volumeCfg := getTestVolumeConfig(t)
		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		expectedHostDir := path.OsAgentDir(volumeCfg.DynakubeName)
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

		pub := NewPublisher(fs, mounter, path)

		resp, err := pub.PublishVolume(ctx, &volumeCfg)
		require.Error(t, err)
		require.Nil(t, resp)
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
