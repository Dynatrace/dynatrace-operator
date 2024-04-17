package csigc

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPathResolver = metadata.PathResolver{
		RootDir: "test",
	}
)

func TestRunSharedImagesGarbageCollection(t *testing.T) {
	ctx := context.TODO()

	t.Run("bad database", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:   fs,
			db:   &metadata.FakeFailDB{},
			path: testPathResolver,
		}
		err := gc.runSharedBinaryGarbageCollection(ctx)
		require.Error(t, err)
	})
	t.Run("no error on empty fs", func(t *testing.T) {
		gc := CSIGarbageCollector{
			fs: afero.NewMemMapFs(),
		}
		err := gc.runSharedBinaryGarbageCollection(ctx)
		require.NoError(t, err)
	})
	t.Run("deletes unused", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:   fs,
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		err := gc.runSharedBinaryGarbageCollection(ctx)
		require.NoError(t, err)
		_, err = fs.Stat(testDir)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("deletes nothing, because of dynakube metadata present", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs: fs,
			db: metadata.FakeMemoryDB(),
		}
		gc.db.CreateTenantConfig(ctx, &metadata.TenantConfig{
			Name:                        "test",
			TenantUUID:                  "test",
			DownloadedCodeModuleVersion: "test",
		})

		err := gc.runSharedBinaryGarbageCollection(ctx)
		require.NoError(t, err)

		_, err = fs.Stat(testDir)
		require.NoError(t, err)
	})
	t.Run("deletes nothing, because of volume metadata present", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs: fs,
			db: metadata.FakeMemoryDB(),
		}
		gc.db.CreateAppMount(ctx, &metadata.AppMount{
			VolumeMeta:        metadata.VolumeMeta{ID: "test", PodName: "test"},
			CodeModuleVersion: testVersion2,
			VolumeMetaID:      "test",
		})

		err := gc.runSharedBinaryGarbageCollection(ctx)
		require.NoError(t, err)

		_, err = fs.Stat(testDir)
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
		dirs, err := gc.getSharedBinDirs()
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
	t.Run("get image cache dirs", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:   fs,
			path: testPathResolver,
		}
		dirs, err := gc.getSharedBinDirs()
		require.NoError(t, err)
		assert.Len(t, dirs, 1)
	})
}

func TestCollectUnusedAgentBins(t *testing.T) {
	ctx := context.TODO()

	t.Run("bad database", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   &metadata.FakeFailDB{},
			path: testPathResolver,
		}
		_, err := gc.collectUnusedAgentBins(ctx, nil)
		require.Error(t, err)
	})
	t.Run("no error on empty db", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		dirs, err := gc.collectUnusedAgentBins(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
	t.Run("get unused", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		testImageDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		testZipDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion1)
		fs := createTestDirs(t, testImageDir, testZipDir)
		imageDirInfo, err := fs.Stat(testImageDir)
		require.NoError(t, err)
		versionDirInfo, err := fs.Stat(testZipDir)
		require.NoError(t, err)

		dirs, err := gc.collectUnusedAgentBins(ctx, []os.FileInfo{imageDirInfo, versionDirInfo})
		require.NoError(t, err)
		assert.Len(t, dirs, 2)
		assert.Equal(t, testImageDir, dirs[0])
	})
	t.Run("gets nothing, image bin is set in dk, zip version is mounted in volume", func(t *testing.T) {
		gc := CSIGarbageCollector{
			db:   metadata.FakeMemoryDB(),
			path: testPathResolver,
		}
		testImageDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		testZipDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion1)

		gc.db.CreateCodeModule(ctx, &metadata.CodeModule{Version: testVersion2, Location: testImageDir})
		gc.db.CreateCodeModule(ctx, &metadata.CodeModule{Version: testVersion1, Location: testZipDir})

		fs := createTestDirs(t, testImageDir, testZipDir)
		imageDirInfo, err := fs.Stat(testImageDir)
		require.NoError(t, err)
		versionDirInfo, err := fs.Stat(testZipDir)
		require.NoError(t, err)

		dirs, err := gc.collectUnusedAgentBins(ctx, []os.FileInfo{imageDirInfo, versionDirInfo})
		require.NoError(t, err)
		assert.Empty(t, dirs)
	})
}

func TestDeleteImageDirs(t *testing.T) {
	t.Run("deletes, no panic/error", func(t *testing.T) {
		testImageDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		testZipDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion1)
		fs := createTestDirs(t, testImageDir, testZipDir)
		err := deleteSharedBinDirs(fs, []string{testImageDir, testZipDir})
		require.NoError(t, err)
		_, err = fs.Stat(testImageDir)
		assert.True(t, os.IsNotExist(err))
		_, err = fs.Stat(testZipDir)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("not exists, no panic/error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion2)
		err := deleteSharedBinDirs(fs, []string{testDir})
		require.NoError(t, err)
	})
}

func createTestDirs(t *testing.T, paths ...string) afero.Fs {
	fs := afero.NewMemMapFs()
	for _, path := range paths {
		require.NoError(t, fs.MkdirAll(path, 0755))
	}

	return fs
}
