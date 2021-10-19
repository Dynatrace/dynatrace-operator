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

func TestGetFieldBool(t *testing.T) {
	data2expectedValid := map[string]bool{
		"true":  true,
		"TRue":  true,
		"yes":   true,
		"yEs":   true,
		"1":     true,
		"false": false,
		"falSE": false,
		"no":    false,
		"nO":    false,
		"0":     false,
	}

	for text, expected := range data2expectedValid {
		t.Run("", func(t *testing.T) {
			m := make(map[string]string)
			m["value"] = text
			value := GetFieldBool(m, "value", true)
			assert.Equal(t, value, expected)
		})
	}

	data2expectedInvalid := []string{"nie", "NEIN", "Tak", "ja", "01", "00", "10", "blabla", ""}
	for _, text := range data2expectedInvalid {
		t.Run("", func(t *testing.T) {
			m := make(map[string]string)
			m["value"] = text
			value1 := GetFieldBool(m, "value", true)
			value2 := GetFieldBool(m, "value", false)

			assert.Equal(t, value1, true)
			assert.Equal(t, value2, false)
		})
	}
}
