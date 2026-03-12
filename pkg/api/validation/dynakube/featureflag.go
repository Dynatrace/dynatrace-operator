package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated = `Feature flag %s is deprecated.\n`
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
}

func deprecatedFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var results strings.Builder

	for _, flag := range deprecatedFeatureFlags {
		if dk.Annotations != nil && dk.Annotations[flag] != "" {
			fmt.Fprintf(&results, warningFeatureFlagDeprecated, flag)
		}
	}

	return results.String()
}
