package address_of

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScalar(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		assert.True(t, *Scalar(true))
		assert.False(t, *Scalar(false))

		const definitelyYes = true
		assert.True(t, *Scalar(definitelyYes))

		probablyYes := true
		assert.True(t, *Scalar(probablyYes))

		probablyYes = false
		assert.False(t, *Scalar(probablyYes))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, int64(23), *Scalar(23))

		const twentySeven = 27
		assert.Equal(t, int64(27), *Scalar(twentySeven))

		roughlyThree := int64(4)
		assert.Equal(t, int64(4), *Scalar(roughlyThree))
	})
}
