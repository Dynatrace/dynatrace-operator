package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated = `Using deprecated feature flags: %s`
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
}

func deprecatedFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var results []string

	for _, flag := range deprecatedFeatureFlags {
		if dk.Annotations != nil && dk.Annotations[flag] != "" {
			results = append(results, flag)
		}
	}

	if len(results) > 0 {
		return fmt.Sprintf(warningFeatureFlagDeprecated, strings.Join(results, ", "))
	}

	return ""
}
