package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testMap = map[string]string{
	"k1": "v1",
	"ke": "",
}

func TestGetField(t *testing.T) {
	t.Run("value present => get value", func(t *testing.T) {
		value := GetField(testMap, "k1", "default")
		assert.Equal(t, "v1", value)
	})
	t.Run("value empty => get default", func(t *testing.T) {
		value := GetField(testMap, "ke", "default")
		assert.Equal(t, "default", value)
	})
	t.Run("value not present => get default", func(t *testing.T) {
		value := GetField(testMap, "other", "default")
		assert.Equal(t, "default", value)
	})
}
