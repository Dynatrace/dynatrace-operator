// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package sanitize provides functions to sanitize user input.
package sanitize

import (
	"os"
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
	sanitized := make([]string, len(args))
	for i, arg := range args {
		sanitized[i] = CommandLineArg(arg)
	}

	return sanitized
}

var filePathSanitizer = strings.NewReplacer("..", "", "//", "/", ",", "", ":", "")

// FilePath removes characters that are unsafe to use as a filesystem path component: "..", "//", ",", and ":".
// Sanitization runs before the root-separator check so that inputs that disguise a bare separator
// (e.g. "../" → "/" after stripping "..") are also stripped.
//
// [filepath.Clean] is not used here because it normalizes a full path rather than stripping characters
// from an untrusted name component. A relative input such as "../secret" passes through Clean unchanged.
func FilePath(path string) string {
	path = filePathSanitizer.Replace(path)

	if path == string(os.PathSeparator) {
		return ""
	}

	return path
}
