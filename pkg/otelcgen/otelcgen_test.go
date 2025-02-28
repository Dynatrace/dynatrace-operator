package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFull(t *testing.T) {
	cfg, err := NewConfig(
		"test",
		WithCA("/run/opensignals/cacerts/certs"),
		WithApiToken("test-token"),
		WithTLS("/run/opensignals/tls/tls.crt", "/run/opensignals/tls/tls.key"),
		WithProtocols("otlp", "jaeger", "zipkin", "statsd"),
		WithExtensions(),
		WithExporters(),
		WithServices("otlp", "jaeger", "zipkin"),
		WithProcessors(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "full_config.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}
