package arg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgFunctions(t *testing.T) {
	t.Run("ArgStringFormat", func(t *testing.T) {
		arg := Arg{Name: "key", Value: "value"}
		expected := "--key=value"
		assert.Equal(t, expected, arg.String())
	})

	t.Run("ConvertArgsToStringsEmpty", func(t *testing.T) {
		args := []Arg{}
		expected := []string{}
		assert.Equal(t, expected, ConvertArgsToStrings(args))
	})

	t.Run("ConvertArgsToStringsSingleArg", func(t *testing.T) {
		args := []Arg{{Name: "key", Value: "value"}}
		expected := []string{"--key=value"}
		assert.Equal(t, expected, ConvertArgsToStrings(args))
	})

	t.Run("ConvertArgsToStringsMultipleArgs", func(t *testing.T) {
		args := []Arg{
			{Name: "key1", Value: "value1"},
			{Name: "key2", Value: "value2"},
		}
		expected := []string{"--key1=value1", "--key2=value2"}
		assert.Equal(t, expected, ConvertArgsToStrings(args))
	})
}
