package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("with statsd protocol only", func(t *testing.T) {
		cfg, err := NewConfig(
			WithPodIP("test"),
			WithProtocols("statsd"),
		)
		require.NoError(t, err)
		c, err := cfg.Marshal()
		require.NoError(t, err)

		expectedOutput, err := os.ReadFile(filepath.Join("testdata", "receivers_statsd.yaml"))
		require.NoError(t, err)

		assert.YAMLEq(t, string(expectedOutput), string(c))
	})

	t.Run("with zipkin protocol only with tls key and tls cert", func(t *testing.T) {
		cfg, err := NewConfig(
			WithPodIP("test"),
			WithTLSCert("/run/opensignals/tls/tls.crt"),
			WithTLSKey("/run/opensignals/tls/tls.key"),
			WithProtocols("zipkin"),
		)
		require.NoError(t, err)
		c, err := cfg.Marshal()
		require.NoError(t, err)

		expectedOutput, err := os.ReadFile(filepath.Join("testdata", "receivers_zipkin_only.yaml"))
		require.NoError(t, err)
		assert.YAMLEq(t, string(expectedOutput), string(c))
	})

	t.Run("with jaeger protocol only with tls key and tls cert", func(t *testing.T) {
		cfg, err := NewConfig(
			WithPodIP("test"),
			WithTLSCert("/run/opensignals/tls/tls.crt"),
			WithTLSKey("/run/opensignals/tls/tls.key"),
			WithProtocols("jaeger"),
		)
		require.NoError(t, err)
		c, err := cfg.Marshal()
		require.NoError(t, err)

		expectedOutput, err := os.ReadFile(filepath.Join("testdata", "receivers_jaeger_only.yaml"))
		require.NoError(t, err)

		assert.YAMLEq(t, string(expectedOutput), string(c))
	})

	t.Run("with otlp protocol only with tls key and tls cert", func(t *testing.T) {
		cfg, err := NewConfig(
			WithPodIP("test"),
			WithTLSCert("/run/opensignals/tls/tls.crt"),
			WithTLSKey("/run/opensignals/tls/tls.key"),
			WithProtocols("otlp"),
		)
		require.NoError(t, err)
		c, err := cfg.Marshal()
		require.NoError(t, err)

		expectedOutput, err := os.ReadFile(filepath.Join("testdata", "receivers_otlp_only.yaml"))
		require.NoError(t, err)

		assert.YAMLEq(t, string(expectedOutput), string(c))
	})

	t.Run("with unknown protocol", func(t *testing.T) {
		_, err := NewConfig(
			WithProtocols("unknown"),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown protocol")
	})
}
