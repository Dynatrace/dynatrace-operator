package deploymentproperties

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func BuildContent(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	keys := slices.Collect(maps.Keys(attrs))
	slices.Sort(keys)

	var sb strings.Builder

	sb.WriteString("[resource_attributes]\n")

	for _, k := range keys {
		fmt.Fprintf(&sb, "%s = %s\n", k, attrs[k])
	}

	return sb.String()
}
