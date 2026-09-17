// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
)

func BuildContent(dk *dynakube.DynaKube) string {
	var sb strings.Builder

	attrs := dk.Spec.ResourceAttributes
	if len(attrs) > 0 {
		keys := slices.Collect(maps.Keys(attrs))
		slices.Sort(keys)

		sb.WriteString("[resource_attributes]\n")

		for _, k := range keys {
			fmt.Fprintf(&sb, "%s = %s\n", k, attrs[k])
		}
	}

	if dk.NeedsCustomNoProxy() {
		noProxyValue := strings.ReplaceAll(dk.FF().GetNoProxy(), ",", "|")
		fmt.Fprintf(&sb, "%s\n%s = %s\n", consts.PropertiesClientInternalSection, consts.PropertiesNoProxyFieldName, noProxyValue)
	}

	return sb.String()
}
