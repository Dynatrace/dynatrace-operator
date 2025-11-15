package zip

import (
	"testing"

	"github.com/klauspost/compress/zip"
	"github.com/stretchr/testify/require"
)

func TestExtractZip(t *testing.T) {
	t.Run("file nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		extractor := createTestExtractor()
		err := extractor.ExtractZip(nil, tmpDir)
		require.EqualError(t, err, "file is nil")
	})
	t.Run("unzip test zip file", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipFile := SetupTestArchive(t, TestRawZip)

		defer func() { _ = zipFile.Close() }()

		fileInfo, err := zipFile.Stat()
		require.NoError(t, err)

		reader, err := zip.NewReader(zipFile, fileInfo.Size())
		require.NoError(t, err)

		err = extractFilesFromZip(tmpDir, reader)
		require.NoError(t, err)
		testUnpackedArchive(t, tmpDir)
	})
}
