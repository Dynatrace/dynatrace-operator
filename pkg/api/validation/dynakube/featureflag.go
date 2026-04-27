package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated = `Feature flag %s is deprecated.`
)

var deprecatedFeatureFlagKeys = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
}

func deprecatedFeatureFlags(_ context.Context, _ *Validator, dk *dynakube.DynaKube) []string {
	var results []string

	for _, flag := range deprecatedFeatureFlagKeys {
		if dk.Annotations != nil && dk.Annotations[flag] != "" {
			results = append(results, fmt.Sprintf(warningFeatureFlagDeprecated, flag))
		}
	}

	return results
}
