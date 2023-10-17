package startup

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	src := "/src"
	err := fs.MkdirAll(src, 0755)
	require.NoError(t, err)

	err = afero.WriteFile(fs, filepath.Join(src, "file1.txt"), []byte("Hello"), 0644)
	require.NoError(t, err)

	err = fs.MkdirAll(filepath.Join(src, "subdir"), 0755)
	require.NoError(t, err)

	err = afero.WriteFile(fs, filepath.Join(src, "subdir", "file2.txt"), []byte("World"), 0644)
	require.NoError(t, err)

	dst := "/dst"
	err = fs.MkdirAll(dst, 0755)
	require.NoError(t, err)

	err = copyFolder(fs, src, dst)
	require.NoError(t, err)

	srcFiles, err := afero.ReadDir(fs, src)
	require.NoError(t, err)
	dstFiles, err := afero.ReadDir(fs, dst)
	require.NoError(t, err)
	require.Len(t, dstFiles, len(srcFiles))

	checkFolder(t, fs, src, dst)
}

func checkFolder(t *testing.T, fs afero.Fs, src, dst string) {
	srcFiles, err := afero.ReadDir(fs, src)
	require.NoError(t, err)
	dstFiles, err := afero.ReadDir(fs, dst)
	require.NoError(t, err)
	require.Len(t, dstFiles, len(srcFiles))

	for i := range srcFiles {
		srcName := srcFiles[i].Name()
		dstName := dstFiles[i].Name()
		require.Equal(t, srcName, dstName)

		srcPath := filepath.Join(src, srcName)
		dstPath := filepath.Join(dst, dstName)

		srcInfo, err := fs.Stat(srcPath)
		require.NoError(t, err)

		dstInfo, err := fs.Stat(dstPath)
		require.NoError(t, err)

		assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())

		if srcInfo.IsDir() {
			assert.True(t, dstInfo.IsDir())
			checkFolder(t, fs, srcPath, dstPath)
		} else {
			srcData, err := afero.ReadFile(fs, srcPath)
			require.NoError(t, err)

			dstData, err := afero.ReadFile(fs, dstPath)
			require.NoError(t, err)

			assert.Equal(t, srcData, dstData)
		}
	}
}
