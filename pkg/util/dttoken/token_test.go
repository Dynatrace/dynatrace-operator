package dttoken

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	t.Run("generate token but always new one", func(t *testing.T) {
		tokenA := New("test")
		tokenB := New("test")

		assert.NotEqual(t, tokenA, tokenB)
	})
	t.Run("string representation of token", func(t *testing.T) {
		tokenA := New("test")
		tokenParts := strings.Split(tokenA.String(), ".")
		assert.Len(t, tokenParts, 3)
		assert.Equal(t, "test", tokenParts[0])
		assert.Len(t, tokenParts[1], publicPortionSize)
		assert.Len(t, tokenParts[2], privatePortionSize)
	})
}

func Test_generateRandom(t *testing.T) {
	t.Run("check random length", func(t *testing.T) {
		assert.Len(t, generateRandom(10), 10)
		assert.Len(t, generateRandom(32), 32)
	})
}
