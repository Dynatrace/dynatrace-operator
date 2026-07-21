// Package sanitize provides functions to sanitize user input.
package sanitize

import (
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
