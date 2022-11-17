package maps

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContainsKey(t *testing.T) {
	testStringMap := map[string]string{
		"key-1": "value",
		"key-2": "value",
		"key-3": "value",
	}

	assert.True(t, ContainsKey(testStringMap, "key-1"))
	assert.True(t, ContainsKey(testStringMap, "key-2"))
	assert.True(t, ContainsKey(testStringMap, "key-3"))

	assert.False(t, ContainsKey(testStringMap, "key-0"))
	assert.False(t, ContainsKey(testStringMap, "key-4"))

	testIntMap := map[int]string{
		-1: "value",
		1:  "value",
		2:  "value",
	}

	assert.True(t, ContainsKey(testIntMap, -1))
	assert.True(t, ContainsKey(testIntMap, 1))
	assert.True(t, ContainsKey(testIntMap, 2))

	assert.False(t, ContainsKey(testIntMap, 0))
	assert.False(t, ContainsKey(testIntMap, 3))
}
