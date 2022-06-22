package csigc

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImageDigest = "5f50f658891613c752d524b72fc"
)

var (
	testPathResolver = metadata.PathResolver{
		RootDir: "test",
	}
)

func TestRunSharedImagesGarbageCollection(t *testing.T) {
	t.Run("bad database", func(t *testing.T) {
		fs := createTestSharedImageDir(t)
		gc := CSIGarbageCollector{
			fs:   fs,
			db:   &metadata.FakeFailDB{},
			path: testPathResolver,
		}
		err := gc.runSharedImagesGarbageCollection()
		require.Error(t, err)
	})
	t.Run("no error on empty fs", func(t *testing.T) {
		gc := CSIGarbageCollector{
			fs: afero.NewMemMapFs(),
		}
		err := gc.runSharedImagesGarbageCollection()
		require.NoError(t, err)
	})
	t.Run("deletes unused", func(t *testing.T) {
		fs := createTestSharedImageDir(t)
		gc := CSIGarbageCollector{
			fs:   fs,
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		err := gc.runSharedImagesGarbageCollection()
		require.NoError(t, err)
		_, err = fs.Stat(gc.path.AgentSharedBinaryDirForImage(testImageDigest))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("deletes nothing", func(t *testing.T) {
		fs := createTestSharedImageDir(t)
		gc := CSIGarbageCollector{
			fs: fs,
			db: metadata.FakeMemoryDB(),
		}
		gc.db.InsertDynakube(&metadata.Dynakube{
			Name:          "test",
			TenantUUID:    "test",
			LatestVersion: "test",
			ImageDigest:   testImageDigest,
		})

		err := gc.runSharedImagesGarbageCollection()
		require.NoError(t, err)

		_, err = fs.Stat(testPathResolver.AgentSharedBinaryDirForImage(testImageDigest))
		require.NoError(t, err)
	})
}

func TestGetSharedImageDirs(t *testing.T) {
	t.Run("no error on empty fs", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		gc := CSIGarbageCollector{
			fs:   fs,
			path: testPathResolver,
		}
		dirs, err := gc.getSharedImageDirs()
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
	t.Run("get image cache dirs", func(t *testing.T) {
		fs := createTestSharedImageDir(t)
		gc := CSIGarbageCollector{
			fs:   fs,
			path: testPathResolver,
		}
		dirs, err := gc.getSharedImageDirs()
		require.NoError(t, err)
		assert.Len(t, dirs, 1)
	})
}

func TestCollectUnusedImageDirs(t *testing.T) {
	t.Run("bad database", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   &metadata.FakeFailDB{},
			path: testPathResolver,
		}
		_, err := gc.collectUnusedImageDirs(nil)
		require.Error(t, err)
	})
	t.Run("no error on empty db", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		dirs, err := gc.collectUnusedImageDirs(nil)
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
	t.Run("get unused", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		fs := createTestSharedImageDir(t)
		testDir := testPathResolver.AgentSharedBinaryDirForImage(testImageDigest)
		fileInfo, err := fs.Stat(testDir)
		require.NoError(t, err)

		dirs, err := gc.collectUnusedImageDirs([]os.FileInfo{fileInfo})
		require.NoError(t, err)
		assert.Len(t, dirs, 1)
		assert.Equal(t, testDir, dirs[0])
	})
	t.Run("gets nothing", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		gc.db.InsertDynakube(&metadata.Dynakube{
			Name:          "test",
			TenantUUID:    "test",
			LatestVersion: "test",
			ImageDigest:   testImageDigest,
		})
		fs := createTestSharedImageDir(t)
		fileInfo, err := fs.Stat(testPathResolver.AgentSharedBinaryDirForImage(testImageDigest))
		require.NoError(t, err)

		dirs, err := gc.collectUnusedImageDirs([]os.FileInfo{fileInfo})
		require.NoError(t, err)
		assert.Len(t, dirs, 0)
	})
}

func TestDeleteImageDirs(t *testing.T) {
	t.Run("deletes, no panic/error", func(t *testing.T) {
		fs := createTestSharedImageDir(t)
		testDir := testPathResolver.AgentSharedBinaryDirForImage(testImageDigest)
		err := deleteImageDirs(fs, []string{testDir})
		require.NoError(t, err)
		_, err = fs.Stat(testDir)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("not exists, no panic/error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testDir := testPathResolver.AgentSharedBinaryDirForImage(testImageDigest)
		err := deleteImageDirs(fs, []string{testDir})
		require.NoError(t, err)
	})
}

func createTestSharedImageDir(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(testPathResolver.AgentSharedBinaryDirForImage(testImageDigest), 0755))
	return fs
}
