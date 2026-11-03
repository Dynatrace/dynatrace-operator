// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package sanitize provides functions to sanitize user input.
package sanitize

import (
	"regexp"
	"strings"
	"sync"
)

var invalidCommandLineChars = [...]rune{
	'\n',
	'\t',
	'\r',
	'\x00',
}

// InvalidCommandLineCharset contains invalid command-line characters.
// Can be used with [strings.ContainsAny].
var InvalidCommandLineCharset = string(invalidCommandLineChars[:])

var argSanitizer = sync.OnceValue(func() *strings.Replacer {
	pairs := make([]string, 0, len(invalidCommandLineChars)*2) //nolint:mnd
	for _, c := range invalidCommandLineChars {
		pairs = append(pairs, string(c), "")
	}

	return strings.NewReplacer(pairs...)
})

// CommandLineArg removes invalid characters from command-line input.
func CommandLineArg(arg string) string {
	return argSanitizer().Replace(arg)
}

// CommandLineArgs returns a copy of args where each element was sanitized with [CommandLineArg].
func CommandLineArgs(args []string) []string {
	if args == nil {
		return nil
	}

	sanitized := make([]string, len(args))
	for i, arg := range args {
		sanitized[i] = CommandLineArg(arg)
	}

	return sanitized
}

var (
	invalidKeyCharsRe      = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	invalidKeyBoundariesRe = regexp.MustCompile(`(?:^[^a-zA-Z0-9]+)|(?:[^a-zA-Z0-9]+$)`)
)

// ResourceAttributeKey returns a sanitized copy of key that is valid as a Kubernetes annotation name segment.
// Characters outside [a-zA-Z0-9\-_.] are replaced with '_', and leading/trailing non-alphanumeric
// characters are stripped. Returns an empty string if nothing valid remains.
func ResourceAttributeKey(key string) string {
	key = invalidKeyCharsRe.ReplaceAllString(key, "_")
	key = invalidKeyBoundariesRe.ReplaceAllString(key, "")

	return key
}
