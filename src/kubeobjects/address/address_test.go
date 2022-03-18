package address

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOf(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		assert.True(t, *Of(true))
		assert.False(t, *Of(false))

		const definitelyYes = true
		assert.True(t, *Of(definitelyYes))

		probablyYes := true
		assert.True(t, *Of(probablyYes))

		probablyYes = false
		assert.False(t, *Of(probablyYes))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, 23, *Of(23))

		const twentySeven = 27
		assert.Equal(t, 27, *Of(twentySeven))

		roughlyThree := int64(4)
		assert.Equal(t, int64(4), *Of(roughlyThree))
	})
}
