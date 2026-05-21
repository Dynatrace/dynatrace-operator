package resourceattributes

import "regexp"

var (
	invalidKeyCharsRe      = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	invalidKeyBoundariesRe = regexp.MustCompile(`^[^a-zA-Z0-9]+|[^a-zA-Z0-9]+$`)
)

// SanitizeKey returns a sanitized copy of key that is valid as a Kubernetes annotation name segment.
// Characters outside [a-zA-Z0-9\-_.] are replaced with '_', and leading/trailing non-alphanumeric
// characters are stripped. Returns an empty string if nothing valid remains.
func SanitizeKey(key string) string {
	key = invalidKeyCharsRe.ReplaceAllString(key, "_")
	key = invalidKeyBoundariesRe.ReplaceAllString(key, "")

	return key
}
