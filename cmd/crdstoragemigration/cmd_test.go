package crdstoragemigration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates command with correct use", func(t *testing.T) {
		cmd := New()
		require.NotNil(t, cmd)
		assert.Equal(t, use, cmd.Use)
	})

	t.Run("has namespace flag", func(t *testing.T) {
		cmd := New()
		require.NotNil(t, cmd)

		flag := cmd.PersistentFlags().Lookup(namespaceFlagName)
		require.NotNil(t, flag)
		assert.Equal(t, namespaceFlagShorthand, flag.Shorthand)
	})
}
