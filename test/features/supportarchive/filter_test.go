package supportarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	isPositive := func(val int) bool {
		return val > 0
	}

	t.Run("empty", func(t *testing.T) {
		input := []int{}
		actual := filter(input, isPositive)
		expected := []int{}

		assert.Equal(t, expected, actual)
	})
	t.Run("none", func(t *testing.T) {
		input := []int{-3, -2, -1}
		actual := filter(input, isPositive)
		expected := []int{}

		assert.Equal(t, expected, actual)
	})
	t.Run("all", func(t *testing.T) {
		input := []int{1, 2, 3}
		actual := filter(input, isPositive)
		expected := []int{1, 2, 3}

		assert.Equal(t, expected, actual)
	})
	t.Run("some", func(t *testing.T) {
		input := []int{-2, -1, 1, 2, -3}
		actual := filter(input, isPositive)
		expected := []int{1, 2}

		assert.Equal(t, expected, actual)
	})
}
