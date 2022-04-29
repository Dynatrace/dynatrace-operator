package zip

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestExtractZip(t *testing.T) {
	t.Run(`file nil`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := ExtractZip(fs, nil, "")
		require.EqualError(t, err, "file is nil")
	})
	t.Run(`unzip test zip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		zipFile := SetupTestArchive(t, fs, TestRawZip)
		defer func() { _ = zipFile.Close() }()

		err := ExtractZip(fs, zipFile, TestZipDirName)
		require.NoError(t, err)

		testUnpackedArchive(t, fs)
	})
}
