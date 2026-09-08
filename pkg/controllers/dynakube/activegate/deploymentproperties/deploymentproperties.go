// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
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
		fmt.Fprintf(&sb, "%s = %s\n", resourceattributes.SanitizeKey(k), sanitize.CommandLineArg(attrs[k]))
	}

	return sb.String()
}
