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
		Protocols{OtlpProtocol, JaegerProtocol, StatsdProtocol, ZipkinProtocol},
		WithCA("/run/opensignals/cacerts/certs"),
		WithSystemCAs(true),
		WithAPIToken("test-token"),
		WithTLS("/run/opensignals/tls/tls.crt", "/run/opensignals/tls/tls.key"),
		WithReceivers(),
		WithExportersEndpoint("exporters-test-endpoint"),
		WithExtensions(),
		WithExporters(),
		WithServices(),
		WithProcessors(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "full_config.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}
