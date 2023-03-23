//go:build e2e

package logs

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertContains(t *testing.T, logStream io.ReadCloser, contains string) {
	buffer := new(bytes.Buffer)
	copied, err := io.Copy(buffer, logStream)

	require.NoError(t, err)
	require.Equal(t, int64(buffer.Len()), copied)
	assert.Contains(t, buffer.String(), contains)
}

func Contains(t *testing.T, logStream io.ReadCloser, contains string) bool {
	buffer := new(bytes.Buffer)
	_, err := io.Copy(buffer, logStream)

	require.NoError(t, err)
	return strings.Contains(buffer.String(), contains)
}
