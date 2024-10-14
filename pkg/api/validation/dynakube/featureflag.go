package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningFeatureFlagDeprecated = `Feature flag %s is deprecated.`
)

func deprecatedFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	results := strings.Builder{}

	for _, flag := range exp.DeprecatedFeatureFlags {
		if dk.Annotations != nil && dk.Annotations[flag] != "" {
			results.WriteString(fmt.Sprintf(warningFeatureFlagDeprecated, flag))
		}
	}

	return results.String()
}
