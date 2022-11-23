package support_archive

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tarFile, err := createTarFile(tmpDir)
	require.NoError(t, err)
	tarball := newTarball(tarFile)

	fileName := tarFile.Name()

	testString := []byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`)
	require.NoError(t, tarball.addFile("lorem-ipsum.txt", bytes.NewReader(testString)))
	tarball.close()
	tarFile.Close()

	resultFile, err := os.OpenFile(fileName, os.O_RDONLY, os.ModeTemporary)
	require.NoError(t, err)
	defer tarFile.Close()

	zipReader, err := gzip.NewReader(resultFile)
	require.NoError(t, err)
	tarReader := tar.NewReader(zipReader)

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "lorem-ipsum.txt", hdr.Name)

	resultString := make([]byte, 1024)
	resultLen, err := tarReader.Read(resultString)
	require.Equal(t, io.EOF, err)
	assert.Equal(t, len(testString), resultLen)
	assert.Equal(t, testString, resultString[:resultLen])

	zipReader.Close()
	resultFile.Close()
}
