package deploymentproperties

import (
	"fmt"
	"slices"
	"strings"
)

func BuildContent(attrs map[string]string) string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	var sb strings.Builder

	sb.WriteString("[resource_attributes]\n")

	for _, k := range keys {
		fmt.Fprintf(&sb, "%s = %s\n", k, attrs[k])
	}

	return sb.String()
}
