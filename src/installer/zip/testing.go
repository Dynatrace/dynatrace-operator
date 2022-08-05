package zip

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestZipDirName  = "test"
	TestZipFilename = "test.txt"

	TestRawZip  = `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`
	TestRawGzip = `H4sIAJovvGIAA+2avW7bMBSFNfsp+ATy5b89BGjQtMhUBEUzFOgiyIwjJLYK/RTu5r1bH7VPUEq1IzlJYQsRGQS83yJDNkDKxDmXR5dXyebSJAtTTNO6KMy6WmRFNDIAoKUkkW7prrCDAiNUMKmBacUZAcaYgIhsxp7Ic9RllRR2KqukSM39fVyaH2mZ3T353cXXT+dfPp+///DtIl8l2Zpcl6YoDx7SQh6ubwQ6I8tscUZBz/icCsEm9k7d3BEcZhQk6AkHsqqylTmjSipp14uxeC6By7lQavLaT4C8hHjqfoy9/rf28/Vl/qu5vvv45zd0HOqfaiZoRKT7qQWv/3h69VABKlNWLsY45v8ghF1/EFoAF0xb/6dUS/R/H5zi/689R8Qdcat6t0XgNP/v6986gET/90Hf/5Ol3f87GGOA/wshVeP/nGr0fx+g/4dN/E/1TgvAAP/f6R841+j/Pni8/4+rzeglwP4fylr8Uf+XjDOqRfP+BwD3/15A/w+b2JnqO/b630b/8X9OH+kftLD6B3dT6ghc/z/zmtzk9XpBqltDTFJWpiBmuUTRh8F+/9ftAtJ8fTPuGMfyX6//o4SGtv/DFdZ/Hwzu/4j5XFIW25WilKqZRKN40+z136jeVQg8Lf/19U+Z4pj/fHCw/l0RKOpNVrXffC/yNH5ZTRie/xTjmP+8gPkvbA70P6rqO4bnP9n2fzH/uQfzX9js+r9OD4Ecy38H5z94+/6XSob13wdY/8Nmp3+nh0BOy399/VOqGeY/Hzzr/yO3A4bnP6aVQP/3Afp/2PT831kTcHj+aw6AY/7zAea/sOnv/1xtAobXf64Fx/rvA6z/YdPXv+vzf9vo1PrPNHCs/z7A+o8gCBImfwHs03GpAEIAAA==`
)

func SetupInvalidTestZip(t *testing.T, fs afero.Fs) afero.File {
	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write([]byte("DIE"))
	require.NoError(t, err)

	return zipFile
}

func SetupTestArchive(t *testing.T, fs afero.Fs, rawZip string) afero.File {
	zipf, err := base64.StdEncoding.DecodeString(rawZip)
	require.NoError(t, err)

	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write(zipf)
	require.NoError(t, err)

	err = zipFile.Sync()
	require.NoError(t, err)

	_, err = zipFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

func createTestExtractor(fs afero.Fs) Extractor {
	return NewOnAgentExtractor(fs, metadata.PathResolver{})
}

func testUnpackedArchive(t *testing.T, fs afero.Fs) {
	exists, err := afero.Exists(fs, filepath.Join(TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, common.AgentConfDirPath, common.RuxitConfFileName))
	require.NoError(t, err)
	assert.True(t, exists)

	info, err := fs.Stat(filepath.Join(TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, common.AgentConfDirPath, common.RuxitConfFileName))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	mode := info.Mode().Perm() & 020
	// Assert file is group writeable
	assert.NotEqual(t, mode, os.FileMode(0))
}
