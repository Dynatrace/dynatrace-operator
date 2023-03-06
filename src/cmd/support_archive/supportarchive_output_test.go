package support_archive

import (
	"archive/tar"
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
	tarBuffer := bytes.Buffer{}

	supportArchiveOutput := "sample output"

	supportArchiveOutputCollector := supportArchiveOutputCollector{
		collectorCommon: collectorCommon{
			log: newSupportArchiveLogger(&logBuffer),
			supportArchive: tarball{
				tarWriter: tar.NewWriter(&tarBuffer),
			},
		},

		output: strings.NewReader(supportArchiveOutput),
	}
	assert.Equal(t, supportArchiveCollectorName, supportArchiveOutputCollector.Name())

	require.NoError(t, supportArchiveOutputCollector.Do())
	tarReader := tar.NewReader(&tarBuffer)

	assert.Contains(t, logBuffer.String(), "Storing support archive output")

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, SupportArchiveOutputFileName, hdr.Name)

	outputFile := make([]byte, hdr.Size)

	bytesRead, err := tarReader.Read(outputFile)
	if !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}
	assert.Equal(t, hdr.Size, int64(bytesRead))
	assert.Equal(t, supportArchiveOutput, string(outputFile))
}
