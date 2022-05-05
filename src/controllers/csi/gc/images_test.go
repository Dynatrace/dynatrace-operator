package csigc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImageDigest = "5f50f658891613c752d524b72fc"
)

func TestGetImageCacheDirs(t *testing.T) {
	t.Run("no error on empty fs", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirs, err := getImageCacheDirs(fs)
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
	t.Run("get image cache dirs", func(t *testing.T) {
		fs := createTestImageCacheDir(t)
		dirs, err := getImageCacheDirs(fs)
		require.NoError(t, err)
		assert.Len(t, dirs, 1)
	})
}

func TestDeleteImageCaches(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		fs := createTestImageCacheDir(t)
		deleteImageCaches(fs, []string{filepath.Join(image.CacheDir, testImageDigest)})
		_, err := fs.Stat(filepath.Join(image.CacheDir, testImageDigest))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestGetRelevantModificationTime(t *testing.T) {
	t.Run("no nil", func(t *testing.T) {
		fs := createTestImageCacheDir(t)
		modTime, err := getRelevantModificationTime(fs, testImageDigest)
		require.NoError(t, err)
		require.NotNil(t, modTime)
	})
}

func createTestImageCacheDir(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll(image.CacheDir, 0755))
	require.NoError(t, fs.Mkdir(filepath.Join(image.CacheDir, testImageDigest), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(image.CacheDir, testImageDigest, "index.json"), []byte("{}"), 0644))
	return fs
}
