package support_archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
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
	require.NoError(t, archive.Close())

	require.NoError(t, archiveFile.Close())

	zipReader, err := zip.OpenReader(fileName)
	require.NoError(t, err)

	assert.Equal(t, "lorem-ipsum.txt", zipReader.File[0].Name)

	outputFile := make([]byte, 1024)
	readCloser, err := zipReader.File[0].Open()
	require.NoError(t, err)

	bytesRead, _ := readCloser.Read(outputFile)
	assert.Len(t, testString, bytesRead)
	assert.Equal(t, testString, outputFile[:bytesRead])

	err = zipReader.Close()
	require.NoError(t, err)
}

func createZipArchiveFile(targetDir string) (*os.File, error) {
	archiveFilePath := fmt.Sprintf(zipArchiveFileName, targetDir, time.Now().Format(time.RFC3339))
	archiveFilePath = strings.ReplaceAll(archiveFilePath, ":", "_")

	tarFile, err := os.Create(archiveFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return tarFile, nil
}
