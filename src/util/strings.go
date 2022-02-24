package util

import "strings"

func RuneIs(wanted rune) func(rune) bool {
	return func(actual rune) bool {
		return actual == wanted
	}
}

func Tokenize(s string, separator rune) []string {
	return strings.FieldsFunc(s, RuneIs(separator))
}
