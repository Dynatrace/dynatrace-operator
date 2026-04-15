package resourceattributes

import "maps"

// MergeResourceAttributes merges base and override maps, with override taking precedence.
// Returns nil when both inputs are nil or empty.
func MergeResourceAttributes(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}

	result := make(map[string]string, len(base)+len(override))

	maps.Copy(result, base)

	maps.Copy(result, override)

	return result
}
