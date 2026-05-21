package zip

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractGzip(t *testing.T) {
	t.Run("path empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		extractor := createTestExtractor()
		err := extractor.ExtractGzip(t.Context(), tmpDir, tmpDir)
		require.Error(t, err)
	})

	t.Run("unzip test gzip file", func(t *testing.T) {
		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, TestRawGzip)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), tmpDir, tarReader)
		require.NoError(t, err)
		testUnpackedArchive(t, tmpDir)
	})

	t.Run("extract gzip with symlinks", func(t *testing.T) {
		// testRawGzipWithSymlinks is a gzip archive containing:
		// - test.txt (regular file with content "you found the easter egg\n")
		// - testdir/ (directory)
		// - testdir/file2.txt (regular file with content "another test file\n")
		// - link_to_test.txt (symlink to test.txt)
		// - link_to_dir (symlink to testdir)
		const testRawGzipWithSymlinks = `H4sIAAMnF2kC/+3ZzWrCQBiF4ax7FXMFcX4zuih02WXvQIJObKgaSMZi774TLLRVpLhICuZ9NkoIBDTfOX4mn+Wzp5fy+BzKdWizQciTa69SGvv9vj+upFY6E8dsBIculm26fDZNei52sd6FR+ULo7RdLFzulVlYVzxkuHsxdDGPxzjkNfqhLuxpxn3hTrOuf8y8UZlySrpCGyvTcWWKNJBCjjn/u7Jdhe0278L7qqvfLs5Lp1XV/X3/H81BVM1hvxbxNYhQdjG0Imw2DP805PQ//X/R/1Z5PScCJtL/67qdDd3/3rnr/Z+m76z/Xfq5IBz9T/6T//+R/+nzt+T/hPK/qrdBD7QI/rn/aX2W/86w/42j3Ddp8WtFfx+I/iZg6tn/6H/6n/6n/+l/+h/0P/0/tf43ynmSYAK29f5tGZvlkM+Bb///10tTZEKP8XCa/3/Jf/L/PP+tNJr8n1D+px1wsGvcnv/O9/uf/lpO2f8AAAAAAAAAAAAAAAAAAPjtEyBMCpgAUAAA`

		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, testRawGzipWithSymlinks)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), tmpDir, tarReader)
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
		require.Equal(t, filepath.Base(testFile), target)

		// Verify directory symlink
		symlinkToDir := filepath.Join(tmpDir, "link_to_dir")
		info, err = os.Lstat(symlinkToDir)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "link_to_dir should be a symlink")

		// Verify directory symlink target
		target, err = os.Readlink(symlinkToDir)
		require.NoError(t, err)
		require.Equal(t, filepath.Base(testDir), target)
	})

	t.Run("extract gzip with nested directories and symlinks", func(t *testing.T) {
		// testRawGzipWithNestedSymlinks is a gzip archive with multiple levels of subdirectories containing symlinks:
		// - root.txt (regular file at root)
		// - level1/ (directory)
		//   - file1.txt (regular file)
		//   - link_to_root.txt (symlink to ../root.txt)
		//   - level2/ (directory)
		//     - file2.txt (regular file)
		//     - link_to_level1.txt (symlink to ../file1.txt)
		//     - link_to_root.txt (symlink to ../../root.txt)
		//     - level3/ (directory)
		//       - file3.txt (regular file)
		//       - link_to_level2.txt (symlink to ../file2.txt)
		//       - link_to_level1_dir (symlink to ../../..)
		const testRawGzipWithNestedSymlinks = `H4sIAJs0F2kC/+2c3Y6aUBRGue5T8ARwzj5/eNGkl73sGxjjYGIGNUE68fHLT9tpxzENjmDqWSsmGCXhAvf6+ADJ8iz/8m11+lqunso6mQQ1cGmplLGv77vPtRItSXpKZuD7sVnV7eaTOJEi3TXbXflZB2+0FDa4TIxRhfefEnh46sOhyZpTM+U2uqH2dpjx4N0w6/LHzIsk2mnlvJH2t9fOv/HKJqmac/53q3pdVlV2LF/Wx+3z2XrtapvNY+7/dLOtynR92DflvmHqoyIj/8n/8/y3QS8wQQRU5UtZ6XzSbfS579zl/G+n703+i/NJ6sj/SPxvzv2v8f8s/g/v+t8oh/4j8n/XAPRUPfCf/U+7N/53bQTQ/2bb/6nuKyATT/+j/9H/WgcXEiw6iCf/+4Xk083/uP5nrQ30P/yP/+/kf7eQgP+j83/XAuTmNXB8//OiHP1vvv4n9D/6H/lP/v/Mf6tFcf9PfPnfL0w+wfyP63/OW03/w//4/079b2E8FwCj9X9XBsztauD4/hdsO5D0v9n6n6H/0f/If/L/d//TC87/Rpv/1Xb/vGwOy+HL5dO2/vD8j+p/Ilr5JJWstVP3Iv8f3P/c/3k3/797/6dVRtA//u8//XARHO9/I14N/p/miiT+5/gf/18+/jc+BE0AxOf/vw78b6XdK47/lbev/tf4H//j/zn9H5wq8H+0/r/tU0Gu8X+7+q/zP9M+ogT/43/8f/b/36Lg/H9M/p9E/Ff7X1mnB/9P/nyqyP0PAAAAAAAAAAAAAAAAAAAAAAAAAP8fPwBLDPj1AHgAAA==`

		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, testRawGzipWithNestedSymlinks)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), tmpDir, tarReader)
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

	t.Run("extract gzip with malicious symlinks blocks dangerous links", func(t *testing.T) {
		// testRawGzipWithMaliciousSymlinks is a gzip archive containing symlinks that attempt to escape the extraction directory:
		// - safe_file.txt (regular file with content "safe content")
		// - testdir/ (directory)
		// - testdir/inner_file.txt (regular file with content "inner content\n")
		// - testdir/evil_relative_symlink (symlink to ../../../etc/passwd - should be blocked)
		// - evil_absolute_symlink (symlink to /etc/passwd - should be blocked)
		// - safe_symlink.txt (symlink to safe_file.txt - should be allowed)
		// - testdir/escape_parent (symlink to ../../outside.txt - should be blocked)
		const testRawGzipWithMaliciousSymlinks = `H4sIANIDG2kC/+3XwWqEMBAG4Jz7FD6BxiTq40iqWQi1Ks64bd++WZfdgqWULcZD/T+EBC85/M7EIXty9cl3LuV3FnHIoDRmWYP1KmVuvvaX97k0eSkSKXYwE9spHC+OiUL+STP07HoWcDjsiFs/ZTHPuBR1VRQ/1/+3fWWUFkmB+t8tf9/3bop0Efze/8tV/1eq0uj/e1hyv10AT+iHR61/d/ZdPbnOsj+7mj5eO9+/7FX/631eFpURiUrT7Po4brLREr21qP9tLbnbZxq6mbfO/e/5G62rkH/M3JH//f//FnusEfDx/HWpw/ynaIfhFP9/1/5PjR1dPdpp+0Hw8fwLmct7/x9mJt/G+gYOnj8AAAAAAAAAAAAA/E+fWFOFnwAoAAA=`

		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, testRawGzipWithMaliciousSymlinks)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), tmpDir, tarReader)
		require.NoError(t, err)

		// Verify safe files were extracted
		safeFile := filepath.Join(tmpDir, "safe_file.txt")
		require.FileExists(t, safeFile)

		testDir := filepath.Join(tmpDir, "testdir")
		require.DirExists(t, testDir)

		innerFile := filepath.Join(testDir, "inner_file.txt")
		require.FileExists(t, innerFile)

		// Verify safe symlink was created
		safeSymlink := filepath.Join(tmpDir, "safe_symlink.txt")
		require.FileExists(t, safeSymlink)

		info, err := os.Lstat(safeSymlink)
		require.NoError(t, err)
		require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "safe_symlink.txt should be a symlink")

		target, err := os.Readlink(safeSymlink)
		require.NoError(t, err)
		require.Equal(t, "safe_file.txt", target)

		// Verify malicious symlinks were NOT created (blocked by security check)
		evilRelativeSymlink := filepath.Join(testDir, "evil_relative_symlink")
		_, err = os.Lstat(evilRelativeSymlink)
		require.True(t, os.IsNotExist(err), "evil_relative_symlink should not exist (should be blocked)")

		evilAbsoluteSymlink := filepath.Join(tmpDir, "evil_absolute_symlink")
		_, err = os.Lstat(evilAbsoluteSymlink)
		require.True(t, os.IsNotExist(err), "evil_absolute_symlink should not exist (should be blocked)")

		// Note: escape_parent symlink (../../outside.txt) actually resolves to a path within tmpDir
		// so it's allowed by the security check. The function checks the final resolved path.
		escapeParent := filepath.Join(testDir, "escape_parent")
		_, err = os.Lstat(escapeParent)
		require.True(t, os.IsNotExist(err), "escape_parent should not exist (should be blocked)")
	})

	t.Run("extract gzip with malicious tar containing a symlink and a file of the same path and filename.", func(t *testing.T) {
		// testRawGzipWithMaliciousSymlinkAndFile is a gzip archive containing a symlink and a file of the same path and filename that attempt to escape the extraction directory
		// testdir/ - directory
		// testdir/escape_parent -> ../../outside.txt - symlink (should be blocked)
		// testdir/escape_parent - regular file

		const testRawGzipWithMaliciousSymlinkAndFile = `H4sICLXb/WkAA2xheWVyLnRhcgDtk00KhDAMRnuUnqBW7c9xpGgWblTaCHP8qTqrDgwMWlHMo5Cu+iV5FCFg13uWExmxWq81ktbveymNVIzrrF19mAM6H+P3vpMOdxNw819AaN0EzeQ8DHhwxurf2j/8m0pbxishinjGGUPfgcDX0Y0tkP9T/BulfvivE/+1KiXjpyzx6f6zfCqCIAji6rwBqGlw6wAMAAA=`

		tmpDir := t.TempDir()

		gzipFile := SetupTestArchive(t, testRawGzipWithMaliciousSymlinkAndFile)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), tmpDir, tarReader)
		require.NoError(t, err)

		// Verify safe symlink was created
		evilFile := filepath.Join(tmpDir, "testdir/escape_parent")
		info, err := os.Lstat(evilFile)
		require.NoError(t, err)
		require.True(t, info.Mode().IsRegular(), "escape_parent should be a regular file (symlink of the same name should be blocked)")
	})

	t.Run("extract gzip with malicious tar containing a hardlink and a file of the same path and filename.", func(t *testing.T) {
		// testRawGzipWithMaliciousSymlinkAndFile is a gzip archive containing a symlink and a file of the same path and filename that attempt to escape the extraction directory
		// testdir/ - directory
		// testdir/escape_parent -> ../../outside.txt - hardlink (should be blocked)
		// testdir/escape_parent - regular file

		const testRawGzipWithMaliciousSymlinkAndFile = `H4sIAAAAAAAAA+2TSwrDIBCGPYonMNr4OE6RZhbZNEEnkOPXPFYGAqUxNGQ+hHHlPzMfIkRs2sBKIhPOmLkm8rq9K2mlZtwU7WpliOhDiv/1nXy4i4CL/wriy/fw7H2ANx6cMft37gv/9mEs40qIKp1uwNg2IHA8urEJ8n+Kf6v1jv86819rJRk/ZYl391/kUxEEQRD/zgdS0LVvAAwAAA==`

		tmpDir := t.TempDir()

		// create a file that is dst file of the hardlink that is part ot the layer.tar
		// to avoid "No such file" error if hardlink is created
		err := os.WriteFile(filepath.Join(tmpDir, "outside.txt"), []byte("outside"), 0600)
		require.NoError(t, err)

		layerTmpDir := filepath.Join(tmpDir, "deeper-tmp")
		err = os.MkdirAll(layerTmpDir, 0755)
		require.NoError(t, err)

		gzipFile := SetupTestArchive(t, testRawGzipWithMaliciousSymlinkAndFile)

		defer func() { _ = gzipFile.Close() }()

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)

		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(t.Context(), layerTmpDir, tarReader)
		require.NoError(t, err)

		// Verify safe symlink was created
		evilFile := filepath.Join(layerTmpDir, "testdir/escape_parent")
		info, err := os.Lstat(evilFile)
		require.NoError(t, err)
		require.True(t, info.Mode().IsRegular(), "escape_parent should be a regular file (symlink of the same name should be blocked)")
	})
}

func TestIsSymlinkSafe(t *testing.T) {
	tests := []struct {
		description string
		targetDir   string
		name        string
		symlink     string
		safe        bool
	}{
		{
			"dir/symlink -> dir/file",
			"/tmpDir",
			"dir/symlink",
			"file",
			true,
		},
		{
			"dir/symlink -> ../file",
			"/tmpDir",
			"dir/symlink",
			"../file",
			true,
		},
		{
			"symlink -> ../file",
			"/tmpDir",
			"symlink",
			"../file",
			false,
		},
		{
			"dir/symlink -> ../../file",
			"/tmpDir",
			"dir/symlink",
			"../../file",
			false,
		},
		{
			"symlink -> /file",
			"/tmpDir",
			"symlink",
			"/file",
			false,
		},
		{
			"symlink -> ..",
			"/tmpDir",
			"symlink",
			"..",
			false,
		},
		{
			"dir/symlink -> ..",
			"/tmpDir",
			"dir/symlink",
			"..",
			true,
		},
		{
			"dir/symlink -> .",
			"/tmpDir",
			"symlink",
			".",
			true,
		},
		{
			"dir/symlink -> subdir/../../file",
			"/tmpDir",
			"dir/symlink",
			"subdir/../../file",
			true,
		},
		{
			"dir/symlink -> subdir/../../../file",
			"/tmpDir",
			"dir/symlink",
			"subdir/../../../file",
			false,
		},
		{
			"dir/symlink -> ''",
			"/tmpDir",
			"symlink",
			"",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			safe := isSymlinkSafeToLink(test.symlink, test.targetDir, evaluateTargetPath(test.targetDir, test.name))
			assert.Equal(t, test.safe, safe)
		})
	}
}

func TestIsHardlinkSafe(t *testing.T) {
	tests := []struct {
		description string
		targetDir   string
		hardlink    string
		safe        bool
	}{
		{
			"hardlink -> file",
			"/tmpDir",
			"file",
			true,
		},
		{
			"hardlink -> dir/file",
			"/tmpDir",
			"dir/file",
			true,
		},
		{
			"hardlink -> ../file",
			"/tmpDir",
			"../file",
			false,
		},
		{
			"hardlink -> ../../file",
			"/tmpDir",
			"../../file",
			false,
		},
		{
			"hardlink -> /file",
			"/tmpDir",
			"/file",
			false,
		},
		{
			"hardlink -> dir/../file",
			"/tmpDir",
			"dir/../b",
			true,
		},
		{
			"hardlink -> dir/../../file",
			"/tmpDir",
			"dir/../../b",
			false,
		},
		{
			"hardlink -> dir/.././../file",
			"/tmpDir",
			"dir/.././../b",
			false,
		},
		{
			"hardlink -> ''",
			"/tmpDir",
			"",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			safe := isHardlinkSafeToLink(test.hardlink, test.targetDir)
			assert.Equal(t, test.safe, safe)
		})
	}
}

func TestIsTargetSafe(t *testing.T) {
	tests := []struct {
		description string
		targetDir   string
		name        string
		safe        bool
	}{
		{
			"file",
			"/tmpDir",
			"file",
			true,
		},
		{
			"/file",
			"/tmpDir",
			"/file",
			true, // always relative to targetDir because target := evaluateTargetPath(targetDir, header.Name)
		},
		{
			"../file",
			"/tmpDir",
			"../file",
			false,
		},
		{
			"/../file",
			"/tmpDir",
			"/../file",
			false,
		},
		{
			"../../file",
			"/tmpDir",
			"../../file",
			false,
		},
		{
			"./file",
			"/tmpDir",
			"./file",
			true,
		},
		{
			"./../file",
			"/tmpDir",
			"./../file",
			false,
		},
		{
			"dir/../../file",
			"/tmpDir",
			"dir/../../file",
			false,
		},
		{
			"empty filename",
			"/tmpDir",
			"",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := isTargetSafeToCreate(test.targetDir, test.name, evaluateTargetPath(test.targetDir, test.name))
			if test.safe {
				assert.NoError(t, err, test.description)
			} else {
				assert.Error(t, err, test.description)
			}
		})
	}
}
