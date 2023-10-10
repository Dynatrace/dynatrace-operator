package zip

import (
	"archive/tar"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestExtractGzip(t *testing.T) {
	t.Run(`path empty`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		extractor := createTestExtractor(fs)
		err := extractor.ExtractGzip("", "")
		require.Error(t, err)
	})

	t.Run(`unzip test gzip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		gzipFile := SetupTestArchive(t, fs, TestRawGzip)

		defer func() { _ = gzipFile.Close() }()

		// afero can't rename directories properly: https://github.com/spf13/afero/issues/141

		reader, err := gzip.NewReader(gzipFile)
		require.NoError(t, err)
		tarReader := tar.NewReader(reader)

		err = extractFilesFromGzip(fs, TestZipDirName, tarReader)
		require.NoError(t, err)
		testUnpackedArchive(t, fs)
	})
}
