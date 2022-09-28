package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
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

	ctx := supportArchiveContext{
		ctx:       context.TODO(),
		targetDir: tmpDir,
	}

	tarball, err := newTarball(&ctx)
	require.NoError(t, err)

	fileName := tarball.tarFile.Name()

	testString := []byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`)
	require.NoError(t, tarball.addFile("lorem-ipsum.txt", bytes.NewReader(testString)))
	tarball.close()

	tarFile, err := os.OpenFile(fileName, os.O_RDONLY, os.ModeTemporary)
	require.NoError(t, err)
	defer tarFile.Close()

	zipReader, err := gzip.NewReader(tarFile)
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
}
