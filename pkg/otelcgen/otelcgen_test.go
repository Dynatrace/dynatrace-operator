package otelcgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("with statsd protocol only", func(t *testing.T) {

		cfg, err := NewConfig(WithProtocols("statsd"))
		require.NoError(t, err)
		c, err := cfg.Marshal()
		require.NoError(t, err)

		expectedOutput, err := os.ReadFile(filepath.Join("testdata", "receivers_statsd.yaml"))
		require.NoError(t, err)

		expected := strings.ReplaceAll(strings.ReplaceAll(string(expectedOutput), "\n", ""), "\r", "")
		actual := strings.ReplaceAll(strings.ReplaceAll(string(c), "\n", ""), "\r", "")
		assert.Equal(t, expected, actual)
	})

}
