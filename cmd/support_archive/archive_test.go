package support_archive

import (
	"archive/zip"
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	archiveFile, err := createZipArchiveFile(tmpDir)
	require.NoError(t, err)

	archive := newZipArchive(archiveFile)

	fileName := archiveFile.Name()

	testString := []byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`)
	require.NoError(t, archive.addFile("lorem-ipsum.txt", bytes.NewReader(testString)))
	assert.NoError(t, archive.Close())

	assert.NoError(t, archiveFile.Close())

	zipReader, err := zip.OpenReader(fileName)
	require.NoError(t, err)

	assert.Equal(t, "lorem-ipsum.txt", zipReader.File[0].Name)

	outputFile := make([]byte, 1024)
	readCloser, err := zipReader.File[0].Open()
	require.NoError(t, err)

	bytesRead, _ := readCloser.Read(outputFile)
	assert.Equal(t, len(testString), bytesRead)
	assert.Equal(t, testString, outputFile[:bytesRead])

	err = zipReader.Close()
	assert.NoError(t, err)
}
