package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCsiCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := newCsiCommandBuilder()
		csiCommand := builder.build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
	t.Run("set config provider", func(t *testing.T) {
		builder := newCsiCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &config.MockProvider{}
		builder = builder.setConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("set manager provider", func(t *testing.T) {
		expectedProvider := &cmdManager.MockProvider{}
		builder := newCsiCommandBuilder().setManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.managerProvider)
	})
	t.Run("set namespace", func(t *testing.T) {
		builder := newCsiCommandBuilder().setNamespace("namespace")

		assert.Equal(t, "namespace", builder.namespace)
	})
	t.Run("set filesystem", func(t *testing.T) {
		_ = newCsiCommandBuilder().getFilesystem()
	})
}
