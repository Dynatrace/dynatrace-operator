package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningFeatureFlagDeprecated   = `Using deprecated feature flags: `
	warningNodeImagePullWithoutCSI = "the `node-image-pull` feature flag only affects the behavior of the CSI driver, other previous `node-image-pull` related behavior has been defaulted."
)

var deprecatedFeatureFlags = []string{
	exp.OAProxyIgnoredKey,   //nolint:staticcheck
	exp.AGUpdatesKey,        //nolint:staticcheck
	exp.AGDisableUpdatesKey, //nolint:staticcheck
	exp.OAMaxUnavailableKey, //nolint:staticcheck
}

func deprecatedFeatureFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var results []string

	if len(dk.Annotations) == 0 {
		return ""
	}

	for _, flag := range deprecatedFeatureFlags {
		if dk.FF().IsSet(flag) {
			results = append(results, flag)
		}
	}

	if len(results) > 0 {
		return warningFeatureFlagDeprecated + strings.Join(results, ", ")
	}

	return ""
}

func isNodeImagePullWithoutCSIDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsCSIAvailable() && dk.FF().IsSet(exp.OANodeImagePullKey) {
		return warningNodeImagePullWithoutCSI
	}

	return ""
}
