package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCollector(t *testing.T) {
	tarBuffer := bytes.Buffer{}
	tarball := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	ctx := supportArchiveContext{
		ctx:           context.TODO(),
		clientSet:     nil,
		apiReader:     nil,
		namespaceName: "",
		toStdout:      false,
		targetDir:     "",
	}

	require.NoError(t, collectOperatorVersion(&ctx, &tarball))
	tarReader := tar.NewReader(&tarBuffer)

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "operator-version.txt", hdr.Name)
}
