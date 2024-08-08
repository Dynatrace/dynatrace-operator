package csigc

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mount "k8s.io/mount-utils"
)

const (
	testVersion = "some-version"
)

var (
	testPathResolver = metadata.PathResolver{
		RootDir: "test",
	}
)

func TestRunBinaryGarbageCollection(t *testing.T) {
	ctx := context.Background()

	t.Run("bad database", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:   fs,
			db:   &metadata.FakeFailDB{},
			path: testPathResolver,
		}
		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.Error(t, err)
	})
	t.Run("no error on empty fs", func(t *testing.T) {
		gc := CSIGarbageCollector{
			fs:      afero.NewMemMapFs(),
			mounter: mount.NewFakeMounter(nil),
			db:      metadata.FakeMemoryDB(),
		}
		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.NoError(t, err)
	})
	t.Run("deletes unused", func(t *testing.T) {
		testSharedDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
		testTenantBinDir := testPathResolver.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		fs := createTestDirs(t, testSharedDir, testTenantBinDir)
		gc := CSIGarbageCollector{
			fs:      fs,
			db:      metadata.FakeMemoryDB(),
			mounter: mount.NewFakeMounter(nil),
			path:    testPathResolver,
		}
		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.NoError(t, err)
		_, err = fs.Stat(testSharedDir)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		_, err = fs.Stat(testTenantBinDir)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("deletes nothing, because of dynakube metadata present", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:      fs,
			db:      metadata.FakeMemoryDB(),
			mounter: mount.NewFakeMounter(nil),
		}
		gc.db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "test",
			TenantUUID:    "test",
			LatestVersion: "test",
			ImageDigest:   testVersion,
		})

		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.NoError(t, err)

		_, err = fs.Stat(testDir)
		require.NoError(t, err)
	})
	t.Run("deletes nothing, because of volume metadata present", func(t *testing.T) {
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
		fs := createTestDirs(t, testDir)
		gc := CSIGarbageCollector{
			fs:      fs,
			db:      metadata.FakeMemoryDB(),
			mounter: mount.NewFakeMounter(nil),
		}
		gc.db.InsertVolume(ctx, &metadata.Volume{
			VolumeID:   "test",
			TenantUUID: "test",
			Version:    testVersion,
			PodName:    "test",
		})

		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.NoError(t, err)

		_, err = fs.Stat(testDir)
		require.NoError(t, err)
	})
	t.Run("deletes nothing, because directory is mounted", func(t *testing.T) {
		testSharedDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
		testTenantBinDir := testPathResolver.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		fs := createTestDirs(t, testSharedDir, testTenantBinDir)
		gc := CSIGarbageCollector{
			fs: fs,
			db: metadata.FakeMemoryDB(),
			mounter: mount.NewFakeMounter([]mount.MountPoint{
				{
					Type: "overlay",
					Opts: []string{"upperdir=beep", "lowerdir=" + testSharedDir, "workdir=boop"},
				},
				{
					Type: "overlay",
					Opts: []string{"lowerdir=" + testTenantBinDir, "upperdir=beep", "workdir=boop"},
				},
			}),
		}

		err := gc.runBinaryGarbageCollection(ctx, testTenantUUID)
		require.NoError(t, err)

		_, err = fs.Stat(testSharedDir)
		require.NoError(t, err)

		_, err = fs.Stat(testTenantBinDir)
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
		testDir := testPathResolver.AgentSharedBinaryDirForAgent(testVersion)
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

func createTestDirs(t *testing.T, paths ...string) afero.Fs {
	fs := afero.NewMemMapFs()
	for _, path := range paths {
		require.NoError(t, fs.MkdirAll(path, 0755))
	}

	return fs
}
