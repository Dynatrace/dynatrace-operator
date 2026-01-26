package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated = `Feature flag %s is deprecated.`
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
}

func deprecatedFeatureFlag(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	results := strings.Builder{}

	for _, flag := range deprecatedFeatureFlags {
		if dk.Annotations != nil && dk.Annotations[flag] != "" {
			results.WriteString(fmt.Sprintf(warningFeatureFlagDeprecated, flag))
		}
	}

	return results.String()
}
