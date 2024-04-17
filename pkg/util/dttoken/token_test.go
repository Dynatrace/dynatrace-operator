package dttoken

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	t.Run("generate token but always new one", func(t *testing.T) {
		tokenA := New("test")
		tokenB := New("test")

		assert.NotEqual(t, tokenA, tokenB)
	})
}
