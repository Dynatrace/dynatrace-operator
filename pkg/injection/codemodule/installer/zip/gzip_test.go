package zip

import (
	"archive/tar"
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
}
