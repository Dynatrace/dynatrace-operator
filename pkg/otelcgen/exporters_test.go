package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWithExporters(t *testing.T) {
	cfg, err := NewConfig(
		"",
		RegisteredProtocols,
		WithExportersEndpoint("test"),
		WithCA("/run/opensignals/cacerts/certs"),
		WithApiToken("test-token"),
		WithExporters(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "exporters_only.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}
