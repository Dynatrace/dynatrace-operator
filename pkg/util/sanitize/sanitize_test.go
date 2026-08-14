// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package sanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidCommandLineCharset(t *testing.T) {
	assert.Equal(t, "\n\t\r\x00", InvalidCommandLineCharset)

	// the charset must work with strings.ContainsAny, which validators rely on
	assert.True(t, strings.ContainsAny("foo\nbar", InvalidCommandLineCharset))
	assert.False(t, strings.ContainsAny("foobar", InvalidCommandLineCharset))
}

func TestCommandLineArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "empty string", in: "", out: ""},
		{name: "clean string is unchanged", in: "foo=bar", out: "foo=bar"},
		{name: "removes newline", in: "foo\nbar", out: "foobar"},
		{name: "removes tab", in: "foo\tbar", out: "foobar"},
		{name: "removes carriage return", in: "foo\rbar", out: "foobar"},
		{name: "removes null", in: "foo\x00bar", out: "foobar"},
		{name: "removes all invalid chars at once", in: "\nfoo\t\rbar\x00", out: "foobar"},
		{name: "removes repeated invalid chars", in: "foo\n\n\nbar", out: "foobar"},
		{name: "keeps regular whitespace", in: "foo bar", out: "foo bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, CommandLineArg(tt.in))
		})
	}
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "empty string", in: "", out: ""},
		{name: "root separator returns empty", in: "/", out: ""},
		{name: "clean path is unchanged", in: "/foo/bar", out: "/foo/bar"},
		{name: "removes double dots", in: "/foo..bar", out: "/foobar"},
		{name: "removes double dots standalone", in: "..", out: ""},
		{name: "converts double slash", in: "/foo//bar", out: "/foo/bar"},
		{name: "removes comma", in: "/foo,bar", out: "/foobar"},
		{name: "removes colon", in: "/foo:bar", out: "/foobar"},
		{name: "removes all invalid patterns at once", in: "/foo:bar,baz//qux..end", out: "/foobarbaz/quxend"},
		{name: "./ is ok", in: "./", out: "./"},
		// inputs that reduce to the path separator after sanitization must also return empty
		{name: "double-dot prefix disguising root", in: "../", out: ""},
		{name: "comma prefix disguising root", in: ",/", out: ""},
		{name: "colon prefix disguising root", in: ":/", out: ""},
		{name: "multiple tricks disguising root", in: ",..:/", out: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, FilePath(tt.in))
		})
	}
}

func TestCommandLineArgs(t *testing.T) {
	t.Run("nil slice returns empty slice", func(t *testing.T) {
		assert.Equal(t, []string{}, CommandLineArgs(nil))
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		assert.Equal(t, []string{}, CommandLineArgs([]string{}))
	})

	t.Run("sanitizes every element", func(t *testing.T) {
		in := []string{"foo\nbar", "clean", "a\tb\x00c"}
		assert.Equal(t, []string{"foobar", "clean", "abc"}, CommandLineArgs(in))
	})

	t.Run("does not mutate the input slice", func(t *testing.T) {
		in := []string{"foo\nbar"}
		_ = CommandLineArgs(in)
		assert.Equal(t, []string{"foo\nbar"}, in)
	})
}
