package zip

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/require"
)

func TestExtractGzip(t *testing.T) {
	t.Run("path empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		extractor := createTestExtractor()
		err := extractor.ExtractGzip(tmpDir, tmpDir)
		require.Error(t, err)
	})

	t.Run("unzip test gzip file", func(t *testing.T) {
		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, TestRawGzip)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(tmpDir, tarReader)
		require.NoError(t, err)
		testUnpackedArchive(t, tmpDir)
	})

	t.Run("extract gzip with symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, TestRawGzipWithSymlinks)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(tmpDir, tarReader)
		require.NoError(t, err)

		// Verify regular files exist
		testFile := filepath.Join(tmpDir, "test.txt")
		require.FileExists(t, testFile)

		testDir := filepath.Join(tmpDir, "testdir")

		file2 := filepath.Join(testDir, "file2.txt")
		require.FileExists(t, file2)

		// Verify symlinks exist and are symlinks
		symlinkToFile := filepath.Join(tmpDir, "link_to_test.txt")
		require.FileExists(t, symlinkToFile)

		info, err := os.Lstat(symlinkToFile)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "link_to_test.txt should be a symlink")

		// Verify symlink target
		target, err := os.Readlink(symlinkToFile)
		require.NoError(t, err)
		require.Equal(t, testFile, target)

		// Verify directory symlink
		symlinkToDir := filepath.Join(tmpDir, "link_to_dir")
		info, err = os.Lstat(symlinkToDir)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "link_to_dir should be a symlink")

		// Verify directory symlink target
		target, err = os.Readlink(symlinkToDir)
		require.NoError(t, err)
		require.Equal(t, testDir, target)
	})
}
