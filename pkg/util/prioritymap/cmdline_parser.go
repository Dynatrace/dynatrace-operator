package prioritymap

import (
	"strings"
)

const DefaultSeparator = "="

// ParseCommandLineArgument splits strings in the format of "--param=value" up in its components "--param", "=" and "value".
// The separator is returned to let the caller know if it was there
func ParseCommandLineArgument(arg string) (string, string, string) {
	arg, value, foundSeparator := strings.Cut(arg, DefaultSeparator)

	if foundSeparator {
		return arg, DefaultSeparator, value
	}

	return arg, "", ""
}
