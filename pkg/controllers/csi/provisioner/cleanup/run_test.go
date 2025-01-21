package cleanup

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestGetFilesystemState(t *testing.T) {
	t.Run("no error on empty FS", func(t *testing.T) {
		cleaner := createCleaner(t)

		fsState, err := cleaner.getFilesystemState()

		require.NoError(t, err)
		assert.Empty(t, fsState)
	})
	t.Run("remove unknown dirs", func(t *testing.T) {
		cleaner := createCleaner(t)

		cleaner.fs.Mkdir(cleaner.path.DynaKubeDir("test1"), os.ModePerm)
		cleaner.fs.Mkdir(cleaner.path.DynaKubeDir("test2"), os.ModePerm)

		files, err := cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		fsState, err := cleaner.getFilesystemState()

		require.NoError(t, err)
		assert.Empty(t, fsState)

		files, err = cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
	t.Run("don't touch unknown files, to keep the db intact, just in case", func(t *testing.T) {
		cleaner := createCleaner(t)

		cleaner.fs.Create(cleaner.path.DynaKubeDir("test1"))
		cleaner.fs.Create(cleaner.path.DynaKubeDir("test2"))

		files, err := cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		fsState, err := cleaner.getFilesystemState()

		require.NoError(t, err)
		assert.Empty(t, fsState)

		files, err = cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})
	t.Run("don't touch well-known dirs", func(t *testing.T) {
		cleaner := createCleaner(t)

		cleaner.fs.Mkdir(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)
		cleaner.fs.Mkdir(cleaner.path.AppMountsBaseDir(), os.ModePerm)

		files, err := cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		fsState, err := cleaner.getFilesystemState()

		require.NoError(t, err)
		assert.Empty(t, fsState)

		files, err = cleaner.fs.ReadDir(cleaner.path.RootDir)
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("get fsState", func(t *testing.T) {
		cleaner := createCleaner(t)

		dkName1 := "dk1"
		dkName2 := "dk2"
		dkName3 := "dk3"

		cleaner.createDeprecatedDirs(t, dkName1, 0, 2)

		cleaner.createBinDirs(t, dkName1)
		cleaner.createBinDirs(t, dkName2)

		cleaner.createHostDirs(t, dkName2)
		cleaner.createHostDirs(t, dkName3)

		fsState, err := cleaner.getFilesystemState()
		require.NoError(t, err)

		assert.Len(t, fsState.deprecatedDks, 1)
		assert.Contains(t, fsState.deprecatedDks, dkName1)

		assert.Len(t, fsState.binDks, 2)
		assert.Contains(t, fsState.binDks, dkName1)
		assert.Contains(t, fsState.binDks, dkName2)

		assert.Len(t, fsState.hostDks, 2)
		assert.Contains(t, fsState.hostDks, dkName2)
		assert.Contains(t, fsState.hostDks, dkName3)
	})
}

func TestSafeAddRelevantPath(t *testing.T) {
	t.Run("no error if path doesn't exist and no addition", func(t *testing.T) {
		cleaner := createCleaner(t)

		relevantPaths := map[string]bool{}

		cleaner.safeAddRelevantPath("something", relevantPaths)
		assert.Empty(t, relevantPaths)
	})

	t.Run("not symlink => added without change", func(t *testing.T) {
		cleaner := createCleaner(t)
		path := "something"
		cleaner.fs.Mkdir(path, os.ModePerm)

		relevantPaths := map[string]bool{}

		cleaner.safeAddRelevantPath(path, relevantPaths)
		assert.Contains(t, relevantPaths, path)
	})

	t.Run("symlink => would be added after following the link", func(t *testing.T) {
		// can't be tested, as it relies on following symlinks
		t.SkipNow()
	})
}

func TestAddRelevantPath(t *testing.T) {
	// can't be tested, as it relies on following symlinks
	t.SkipNow()
}

func createCleaner(t *testing.T) *Cleaner {
	t.Helper()

	return &Cleaner{
		fs:        afero.Afero{Fs: afero.NewMemMapFs()},
		mounter:   mount.NewFakeMounter(nil),
		apiReader: fake.NewClient(),
		path:      metadata.PathResolver{},
	}
}

func (c *Cleaner) createBinDirs(t *testing.T, name string) {
	t.Helper()

	binDir := c.path.LatestAgentBinaryForDynaKube(name)
	err := c.fs.MkdirAll(binDir, os.ModePerm)
	require.NoError(t, err)
}

func (c *Cleaner) createHostDirs(t *testing.T, name string) {
	t.Helper()

	hostDir := c.path.OsAgentDir(name)
	err := c.fs.MkdirAll(hostDir, os.ModePerm)
	require.NoError(t, err)
}
