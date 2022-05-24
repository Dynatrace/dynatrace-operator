package address

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOf(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		assert.True(t, *Of(true))
		assert.False(t, *Of(false))

		const constantBool = true
		assert.True(t, *Of(constantBool))

		mutableBool := true
		assert.True(t, *Of(mutableBool))

		mutableBool = false
		assert.False(t, *Of(mutableBool))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, 23, *Of(23))

		const constantInt = 27
		assert.Equal(t, 27, *Of(constantInt))

		mutableInt := int64(4)
		assert.Equal(t, int64(4), *Of(mutableInt))
	})
}
