package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCsiCommandBuilder(t *testing.T) {
	t.Run("set config provider", func(t *testing.T) {
		builder := newCsiCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &config.MockProvider{}
		builder = builder.setConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("build command", func(t *testing.T) {
		builder := newCsiCommandBuilder()
		csiCommand := builder.build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
}
