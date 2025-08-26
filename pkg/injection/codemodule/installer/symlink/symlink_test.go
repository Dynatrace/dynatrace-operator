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

		info, err := fs.Stat(testPath)
		require.Nil(t, info)
		require.ErrorIs(t, err, os.ErrNotExist)
	})
	t.Run("silence error if not present", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		err := Remove(fs, testPath)
		require.NoError(t, err)
	})
}
