package supportarchive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuppotrArchiveOutputCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}
	buffer := bytes.Buffer{}

	supportArchiveOutput := "sample output"

	archive := newZipArchive(bufio.NewWriter(&buffer))
	supportArchiveOutputCollector := supportArchiveOutputCollector{
		collectorCommon: collectorCommon{
			log:            newSupportArchiveLogger(&logBuffer),
			supportArchive: archive,
		},

		output: strings.NewReader(supportArchiveOutput),
	}
	assert.Equal(t, supportArchiveCollectorName, supportArchiveOutputCollector.Name())

	require.NoError(t, supportArchiveOutputCollector.Do())
	assertNoErrorOnClose(t, archive)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))

	assert.Contains(t, logBuffer.String(), "Storing support archive output")

	require.NoError(t, err)
	require.Len(t, zipReader.File, 1)
	assert.Equal(t, SupportArchiveOutputFileName, zipReader.File[0].Name)

	size := zipReader.File[0].FileInfo().Size()
	outputFile := make([]byte, size)

	readCloser, err := zipReader.File[0].Open()
	require.NoError(t, err)

	bytesRead, err := readCloser.Read(outputFile)
	if !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}

	assert.Equal(t, size, int64(bytesRead))
	assert.Equal(t, supportArchiveOutput, string(outputFile))
}
