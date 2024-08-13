package csigc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mount "k8s.io/mount-utils"
)

const (
	testRootDir    = "root-dir"
	testTenantUUID = "tenant-12"

	testVersion1 = "v1"
	testVersion2 = "v2"
	testVersion3 = "v3"
)

var (
	testVolumeFolderPath = filepath.Join(testRootDir, testTenantUUID, "run")
)

func TestGetUnmountedVolumes(t *testing.T) {
	t.Run("no error if no volumes are present", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		_ = gc.fs.MkdirAll(testVolumeFolderPath, 0770)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)

		require.NoError(t, err)
		assert.Equal(t, []os.FileInfo(nil), unmountedVolumes)
	})
	t.Run("mounted volumes are not collected", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		gc.mockMountedVolumeIDPath(testVersion1)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)

		require.NoError(t, err)
		assert.Equal(t, []os.FileInfo(nil), unmountedVolumes)
	})
	t.Run("unmounted volumes are collected", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		gc.mockUnmountedVolumeIDPath(testVersion1)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)

		require.NoError(t, err)
		assert.Equal(t, testVersion1, unmountedVolumes[0].Name())
	})
	t.Run("multiple unmounted volumes are collected", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2, testVersion3)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)

		require.NoError(t, err)
		require.Len(t, unmountedVolumes, 3)
		assert.Equal(t, testVersion1, unmountedVolumes[0].Name())
		assert.Equal(t, testVersion2, unmountedVolumes[1].Name())
		assert.Equal(t, testVersion3, unmountedVolumes[2].Name())
	})
	t.Run("multiple unmounted volumes are collected while mounted volume is present", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		gc.mockMountedVolumeIDPath(testVersion3)
		gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)

		require.NoError(t, err)
		require.Len(t, unmountedVolumes, 2)
		assert.Equal(t, testVersion1, unmountedVolumes[0].Name())
		assert.Equal(t, testVersion2, unmountedVolumes[1].Name())
	})
}

func TestIsUnmountedVolumeTooOld(t *testing.T) {
	t.Run("default is false for current timestamp", func(t *testing.T) {
		gc := CSIGarbageCollector{
			maxUnmountedVolumeAge: defaultMaxUnmountedCsiVolumeAge,
		}

		isOlder := gc.isUnmountedVolumeTooOld(time.Now())

		assert.False(t, isOlder)
	})

	t.Run("default is true for timestamp 14 days in past", func(t *testing.T) {
		gc := CSIGarbageCollector{
			maxUnmountedVolumeAge: defaultMaxUnmountedCsiVolumeAge,
		}

		isOlder := gc.isUnmountedVolumeTooOld(time.Now().AddDate(0, 0, -15))

		assert.True(t, isOlder)
	})

	t.Run("is true if maxUnmountedCsiVolumeAge == 0", func(t *testing.T) {
		gc := CSIGarbageCollector{
			maxUnmountedVolumeAge: 0,
		}

		isOlder := gc.isUnmountedVolumeTooOld(time.Now().AddDate(0, 0, 15))

		assert.True(t, isOlder)
	})
}

func TestRemoveUnmountedVolumesIfNecessary(t *testing.T) {
	t.Run("remove only too old unmounted volume", func(t *testing.T) {
		gc := NewMockGarbageCollector()
		gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2, testVersion3)

		unmountedVolumes, err := gc.getUnmountedVolumes(testTenantUUID)
		require.NoError(t, err)
		require.NotNil(t, unmountedVolumes)

		oldVolume := unmountedVolumes[0]
		err = gc.fs.Chtimes(filepath.Join(testVolumeFolderPath, oldVolume.Name()), oldVolume.ModTime(), oldVolume.ModTime().AddDate(0, 0, -15))
		require.NoError(t, err)

		older := gc.isUnmountedVolumeTooOld(oldVolume.ModTime())
		require.True(t, older)

		gc.removeUnmountedVolumesIfNecessary(unmountedVolumes, testTenantUUID)
		oldVolumeExists, err := afero.DirExists(gc.fs, filepath.Join(testVolumeFolderPath, oldVolume.Name()))
		require.NoError(t, err)
		assert.False(t, oldVolumeExists)

		for _, remainingVolume := range unmountedVolumes[1:] {
			volumeExists, err := afero.DirExists(gc.fs, filepath.Join(testVolumeFolderPath, remainingVolume.Name()))
			require.NoError(t, err)
			assert.True(t, volumeExists)
		}
	})
}

func TestDetermineMaxUnmountedVolumeAge(t *testing.T) {
	t.Run("no env set ==> use default", func(t *testing.T) {
		maxVolumeAge := determineMaxUnmountedVolumeAge("")

		assert.Equal(t, defaultMaxUnmountedCsiVolumeAge, maxVolumeAge)
	})

	t.Run("use (short) duration from env", func(t *testing.T) {
		maxVolumeAge := determineMaxUnmountedVolumeAge("1")

		assert.Equal(t, time.Hour*24, maxVolumeAge)
	})

	t.Run("negative duration in env => use 0", func(t *testing.T) {
		maxVolumeAge := determineMaxUnmountedVolumeAge("-1")

		assert.Equal(t, time.Duration(0), maxVolumeAge)
	})
}

func (gc *CSIGarbageCollector) mockMountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(testVolumeFolderPath, volumeID, "mapped", "something"), os.ModePerm)
	}
}

func (gc *CSIGarbageCollector) mockUnmountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(testVolumeFolderPath, volumeID, "mapped"), os.ModePerm)
	}
}

func NewMockGarbageCollector(mountPoints ...mount.MountPoint) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		fs:                    afero.NewMemMapFs(),
		db:                    metadata.FakeMemoryDB(),
		path:                  metadata.PathResolver{RootDir: testRootDir},
		mounter:               mount.NewFakeMounter(mountPoints),
		maxUnmountedVolumeAge: defaultMaxUnmountedCsiVolumeAge,
	}
}
