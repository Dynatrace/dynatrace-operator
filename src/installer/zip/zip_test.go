package zip

import (
	"testing"

	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestExtractZip(t *testing.T) {
	t.Run(`file nil`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		extractor := createTestExtractor(fs)
		err := extractor.ExtractZip(nil, "")
		require.EqualError(t, err, "file is nil")
	})
	t.Run(`unzip test zip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		zipFile := SetupTestArchive(t, fs, TestRawZip)

		defer func() { _ = zipFile.Close() }()

		// afero can't rename directories properly: https://github.com/spf13/afero/issues/141
		// extractor := createTestExtractor(fs)
		// err := extractor.ExtractZip(zipFile, TestZipDirName)
		// require.NoError(t, err)

		fileInfo, err := zipFile.Stat()
		require.NoError(t, err)

		reader, err := zip.NewReader(zipFile, fileInfo.Size())
		require.NoError(t, err)

		err = extractFilesFromZip(fs, TestZipDirName, reader)
		require.NoError(t, err)
		testUnpackedArchive(t, fs)
	})
}
