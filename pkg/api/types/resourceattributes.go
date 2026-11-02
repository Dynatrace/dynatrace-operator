// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"maps"
	"regexp"
)

// MergeResourceAttributes merges base and override maps, with override taking precedence.
// Returns nil when both inputs are nil or empty.
func MergeResourceAttributes(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}

	result := make(map[string]string)

	maps.Copy(result, base)

	maps.Copy(result, override)

	return result
}

var (
	invalidKeyCharsRe      = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	invalidKeyBoundariesRe = regexp.MustCompile(`(?:^[^a-zA-Z0-9]+)|(?:[^a-zA-Z0-9]+$)`)
)

// SanitizeResourceAttributeKey returns a sanitized copy of key that is valid as a Kubernetes annotation name segment.
// Characters outside [a-zA-Z0-9\-_.] are replaced with '_', and leading/trailing non-alphanumeric
// characters are stripped. Returns an empty string if nothing valid remains.
func SanitizeResourceAttributeKey(key string) string {
	key = invalidKeyCharsRe.ReplaceAllString(key, "_")
	key = invalidKeyBoundariesRe.ReplaceAllString(key, "")

	return key
}
