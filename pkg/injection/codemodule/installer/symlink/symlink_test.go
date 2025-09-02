package symlink

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindVersionFromFileSystem(t *testing.T) {
	testPath := "/test"
	testVersion := "1.239.14.20220325-164521"

	t.Run("get version from directory in file system", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := fs.MkdirAll(filepath.Join(testPath, testVersion), 0755)
		require.NoError(t, err)

		version, err := findVersionFromFileSystem(fs, testPath)
		require.NoError(t, err)
		assert.Equal(t, testVersion, version)
	})
	t.Run("get nothing from file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := fs.MkdirAll(testPath, 0755)
		require.NoError(t, err)
		_, err = fs.Create(filepath.Join(testPath, testVersion))
		require.NoError(t, err)

		version, err := findVersionFromFileSystem(fs, testPath)
		require.NoError(t, err)
		assert.Empty(t, version)
	})
}

func TestRemove(t *testing.T) {
	testPath := "/test/1.239.14.20220325-164521"

	t.Run("removes if present -> no error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := fs.MkdirAll(testPath, 0755)
		require.NoError(t, err)

		err = Remove(fs, testPath)
		require.NoError(t, err)

		entries, err := afero.Afero{Fs: fs}.ReadDir(filepath.Dir(testPath))
		require.NoError(t, err)
		require.Empty(t, entries)
	})
	t.Run("silence error if not present", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		err := Remove(fs, testPath)
		require.NoError(t, err)
	})

	t.Run("works for dangling symlink", func(t *testing.T) {
		base := t.TempDir()
		fs := afero.NewOsFs()

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

		err = Remove(fs, danglingLink)
		require.NoError(t, err)

		entries, err = os.ReadDir(base)
		require.NoError(t, err)
		require.Empty(t, entries)
	})
}
