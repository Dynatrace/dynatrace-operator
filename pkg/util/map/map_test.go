package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testValue1 = "test-value"
	testValue2 = "test-alternative-value"
	testKey1   = "test-key"
	testKey2   = "test-name"
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

func TestGetFieldBool(t *testing.T) {
	data2expectedValid := map[string]bool{
		"true":  true,
		"1":     true,
		"false": false,
		"FALSE": false,
		"0":     false,
	}

	for text, expected := range data2expectedValid {
		t.Run("", func(t *testing.T) {
			m := make(map[string]string)
			m["value"] = text
			value := GetFieldBool(m, "value", true)
			assert.Equal(t, expected, value)
		})
	}

	data2expectedInvalid := []string{"nie", "NEIN", "Tak", "ja", "01", "00", "10", "blabla", ""}
	for _, text := range data2expectedInvalid {
		t.Run("", func(t *testing.T) {
			m := make(map[string]string)
			m["value"] = text
			value1 := GetFieldBool(m, "value", true)
			value2 := GetFieldBool(m, "value", false)

			assert.True(t, value1)
			assert.False(t, value2)
		})
	}
}

func TestMergeMap(t *testing.T) {
	initialMap := map[string]string{
		testKey1: testValue1,
	}
	expectedMap := map[string]string{
		testKey1: testValue1,
		testKey2: testValue2,
	}

	assert.Equal(t, expectedMap, MergeMap(initialMap, map[string]string{
		testKey2: testValue2,
	}))
}
