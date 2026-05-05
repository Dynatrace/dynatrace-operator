package deploymentproperties

import (
	"fmt"
	"sort"
	"strings"
)

func BuildContent(attrs map[string]string) string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var sb strings.Builder

	sb.WriteString("[resource_attributes]\n")

	for _, k := range keys {
		fmt.Fprintf(&sb, "%s = %s\n", k, attrs[k])
	}

	return sb.String()
}
