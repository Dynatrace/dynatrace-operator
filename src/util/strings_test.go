package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneIs(t *testing.T) {
	t.Run("positive matches", func(t *testing.T) {
		assert.True(t, RuneIs('ą')('ą'))
		assert.True(t, RuneIs('\t')('	'))
		assert.True(t, RuneIs(' ')(' '))
		assert.True(t, RuneIs(0)(0))
	})

	t.Run("negative matches", func(t *testing.T) {
		assert.False(t, RuneIs('ą')('ä'))
		assert.False(t, RuneIs('\t')(' '))
		assert.False(t, RuneIs(' ')('	'))
		assert.False(t, RuneIs(0)(1))
	})
}

func TestTokenize(t *testing.T) {
	t.Run("separators only", func(t *testing.T) {
		assert.Equal(t, []string{}, Tokenize("", ' '))
		assert.Equal(t, []string{}, Tokenize(" ", ' '))
		assert.Equal(t, []string{}, Tokenize("      ", ' '))
	})

	t.Run("start or end with separators", func(t *testing.T) {
		assert.Equal(t, []string{"aaa"}, Tokenize("#aaa", '#'))
		assert.Equal(t, []string{"aaa"}, Tokenize("aaa#", '#'))
		assert.Equal(t, []string{"aaa"}, Tokenize("#aaa#", '#'))
		assert.Equal(t, []string{"aaa"}, Tokenize("####aaa####", '#'))
	})

	t.Run("merged separators", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b", "c"}, Tokenize("/a/b/c", '/'))
		assert.Equal(t, []string{"a", "b", "c"}, Tokenize("/a/b/c/", '/'))
		assert.Equal(t, []string{"a", "b", "c"}, Tokenize("a/b/c", '/'))
		assert.Equal(t, []string{"a", "b", "c"}, Tokenize("//a//b//c", '/'))
		assert.Equal(t, []string{"a", "b", "c"}, Tokenize("//a//b//c//", '/'))
	})
}
