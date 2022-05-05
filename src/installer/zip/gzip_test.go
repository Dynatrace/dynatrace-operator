package zip

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestExtractGzip(t *testing.T) {
	t.Run(`path empty`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := ExtractGzip(fs, "", "")
		require.Error(t, err)
	})
	t.Run(`unzip test gzip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		gzipFile := SetupTestArchive(t, fs, TestRawGzip)
		defer func() { _ = gzipFile.Close() }()

		err := ExtractGzip(fs, gzipFile.Name(), TestZipDirName)
		require.NoError(t, err)

		testUnpackedArchive(t, fs)
	})
}
