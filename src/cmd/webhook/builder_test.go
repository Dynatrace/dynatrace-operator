package webhook

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/stretchr/testify/assert"
)

func TestWebhookCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := newWebhookCommandBuilder()
		csiCommand := builder.build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
	t.Run("set config provider", func(t *testing.T) {
		builder := newWebhookCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &config.MockProvider{}
		builder = builder.setConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("set manager provider", func(t *testing.T) {
		expectedProvider := &cmdManager.MockProvider{}
		builder := newWebhookCommandBuilder().setManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.managerProvider)
	})
	t.Run("set namespace", func(t *testing.T) {
		builder := newWebhookCommandBuilder().setNamespace("namespace")

		assert.Equal(t, "namespace", builder.namespace)
	})
	t.Run("set deployed via olm flag", func(t *testing.T) {
		builder := newWebhookCommandBuilder().setIsDeployedViaOlm(true)

		assert.True(t, builder.isDeployedViaOlm)

		builder = builder.setIsDeployedViaOlm(false)
		assert.False(t, builder.isDeployedViaOlm)
	})
}
