package address_of

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	assert.True(t, *Bool(true))
	assert.False(t, *Bool(false))

	const definitelyYes = true
	assert.True(t, *Bool(definitelyYes))

	probablyYes := true
	assert.True(t, *Bool(probablyYes))

	probablyYes = false
	assert.False(t, *Bool(probablyYes))
}

func TestInt64(t *testing.T) {
	assert.Equal(t, int64(23), *Int64(23))

	const twentySeven = 27
	assert.Equal(t, int64(27), *Int64(twentySeven))

	roughlyThree := int64(4)
	assert.Equal(t, int64(4), *Int64(roughlyThree))
}
