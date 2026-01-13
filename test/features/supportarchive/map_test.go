package supportarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("map slice of struct to slice of strings", func(t *testing.T) {
		type Foo struct {
			S string
		}

		foos := []Foo{{S: "aaa"}, {S: "bbb"}}
		actual := Map(foos, func(f Foo) string {
			return f.S
		})
		expected := []string{"aaa", "bbb"}

		assert.Equal(t, expected, actual)
	})
}
