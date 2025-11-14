package zip

import (
	"archive/tar"
	"os"
	"path/filepath"
	"runtime"
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
		if runtime.GOOS == "darwin" {
			t.Skip()
		}
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

	t.Run("extract gzip with nested directories and symlinks", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip()
		}

		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, TestRawGzipWithNestedSymlinks)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(tmpDir, tarReader)
		require.NoError(t, err)

		// Verify root file
		rootFile := filepath.Join(tmpDir, "root.txt")
		require.FileExists(t, rootFile)

		// Verify level 1
		level1File := filepath.Join(tmpDir, "level1", "file1.txt")
		require.FileExists(t, level1File)

		level1Symlink := filepath.Join(tmpDir, "level1", "link_to_root.txt")
		info, err := os.Lstat(level1Symlink)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "level1/link_to_root.txt should be a symlink")

		target, err := os.Readlink(level1Symlink)
		require.NoError(t, err)
		require.Contains(t, target, "root.txt")

		// Verify level 2
		level2File := filepath.Join(tmpDir, "level1", "level2", "file2.txt")
		require.FileExists(t, level2File)

		level2SymlinkToLevel1 := filepath.Join(tmpDir, "level1", "level2", "link_to_level1.txt")
		info, err = os.Lstat(level2SymlinkToLevel1)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "level2/link_to_level1.txt should be a symlink")

		target, err = os.Readlink(level2SymlinkToLevel1)
		require.NoError(t, err)
		require.Contains(t, target, "file1.txt")

		level2SymlinkToRoot := filepath.Join(tmpDir, "level1", "level2", "link_to_root.txt")
		info, err = os.Lstat(level2SymlinkToRoot)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "level2/link_to_root.txt should be a symlink")

		target, err = os.Readlink(level2SymlinkToRoot)
		require.NoError(t, err)
		require.Contains(t, target, "root.txt")

		// Verify level 3
		level3File := filepath.Join(tmpDir, "level1", "level2", "level3", "file3.txt")
		require.FileExists(t, level3File)

		level3SymlinkToLevel2 := filepath.Join(tmpDir, "level1", "level2", "level3", "link_to_level2.txt")
		info, err = os.Lstat(level3SymlinkToLevel2)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "level3/link_to_level2.txt should be a symlink")

		target, err = os.Readlink(level3SymlinkToLevel2)
		require.NoError(t, err)
		require.Contains(t, target, "file2.txt")

		level3DirSymlink := filepath.Join(tmpDir, "level1", "level2", "level3", "link_to_level1_dir")
		info, err = os.Lstat(level3DirSymlink)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "level3/link_to_level1_dir should be a symlink")
	})
}
