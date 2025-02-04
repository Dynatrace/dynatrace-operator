package support_archive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}

	buffer := bytes.Buffer{}

	archive := newZipArchive(bufio.NewWriter(&buffer))
	versionCollector := operatorVersionCollector{
		collectorCommon{
			log:            newSupportArchiveLogger(&logBuffer),
			supportArchive: archive,
		},
	}
	assert.Equal(t, operatorVersionCollectorName, versionCollector.Name())

	require.NoError(t, versionCollector.Do())

	err := archive.Close()
	require.NoError(t, err)
	assert.Contains(t, logBuffer.String(), "Storing operator version")

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	assert.Len(t, zipReader.File, 1)
	file := zipReader.File[0]
	assert.Equal(t, "operator-version.txt", file.Name)

	size := file.FileInfo().Size()
	versionFile := make([]byte, size)
	reader, err := file.Open()
	bytesRead, _ := reader.Read(versionFile)

	if !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}

	assert.Equal(t, size, int64(bytesRead))
}
