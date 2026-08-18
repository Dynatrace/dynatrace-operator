// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package sanitize

import (
	"slices"
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

func TestCommandLineArgs(t *testing.T) {
	tests := []struct {
		name   string
		in     []string
		expect []string
	}{
		{"preserve nil", nil, nil},
		{"empty slice", []string{}, []string{}},
		{"sanitize", []string{"foo\nbar", "clean", "a\tb\x00c"}, []string{"foobar", "clean", "abc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := slices.Clone(tt.in)
			assert.Equal(t, tt.expect, CommandLineArgs(tt.in))
			assert.Equal(t, old, tt.in, "input slice was mutated")
		})
	}
}
