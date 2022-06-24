package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigProvider(t *testing.T) {
	provider := newKubeConfigProvider()

	assert.NotNil(t, provider)
}
