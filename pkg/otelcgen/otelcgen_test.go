package otelcgen

import (
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg, err := NewConfig(WithProtocols())
	require.NoError(t, err)

	c, err := cfg.Marshal()
	require.NoError(t, err)

	assert.Equal(t, "{}", string(c))
}
