package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWithProcessors(t *testing.T) {
	cfg, err := NewConfig(
		"",
		RegisteredProtocols,
		WithProcessors(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "processors_only.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}
