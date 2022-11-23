package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}

	tarBuffer := bytes.Buffer{}

	ctx := supportArchiveContext{
		ctx:           context.TODO(),
		namespaceName: "",
		log:           newSupportArchiveLoggerWithWriter(&logBuffer),
		supportArchive: tarball{
			tarWriter: tar.NewWriter(&tarBuffer),
		},
	}

	require.NoError(t, collectOperatorVersion(ctx))
	tarReader := tar.NewReader(&tarBuffer)

	assert.Contains(t, logBuffer.String(), "Storing operator version")

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "operator-version.txt", hdr.Name)

	versionFile := make([]byte, hdr.Size)

	bytesRead, err := tarReader.Read(versionFile)
	if err != io.EOF {
		require.NoError(t, err)
	}
	assert.Equal(t, hdr.Size, int64(bytesRead))
}
