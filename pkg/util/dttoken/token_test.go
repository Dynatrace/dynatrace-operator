package dttoken

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	t.Run("generate token but always new one", func(t *testing.T) {
		tokenA, err := New("test")
		require.NoError(t, err)
		tokenB, err := New("test")
		require.NoError(t, err)

		assert.NotEqual(t, tokenA, tokenB)
	})
	t.Run("string representation of token", func(t *testing.T) {
		tokenA, err := New("test")
		require.NoError(t, err)

		tokenParts := strings.Split(tokenA.String(), ".")
		assert.Len(t, tokenParts, 3)
		assert.Equal(t, "test", tokenParts[0])
		assert.Len(t, tokenParts[1], publicPortionSize)
		assert.Len(t, tokenParts[2], privatePortionSize)
	})
	t.Run("string prefix including space", func(t *testing.T) {
		token, err := New("EEC dt0x01")
		require.NoError(t, err)

		require.Equal(t, "EEC dt0x01", token.prefix)
		require.NotEmpty(t, token.private)
		require.NotEmpty(t, token.public)
	})
}

func Test_generateRandom(t *testing.T) {
	t.Run("check random length", func(t *testing.T) {
		ten, err := generateRandom(10)
		require.NoError(t, err)
		assert.Len(t, ten, 10)
	})
	t.Run("error", func(t *testing.T) {
		r, err := generateRandom(32)
		require.NoError(t, err)
		require.Len(t, r, 32)
	})
}
