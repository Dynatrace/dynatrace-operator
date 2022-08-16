package troubleshoot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTroubleshootCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewTroubleshootCommandBuilder()
		csiCommand := builder.Build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
}
