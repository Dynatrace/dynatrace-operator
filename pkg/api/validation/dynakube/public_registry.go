package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
)

const (
	errorPublicRegistryOverrideWithoutPublicRegistry = `The publicRegistryOverride field is set, but the feature flag "%s" is not enabled. Either enable the feature flag or remove the publicRegistryOverride field.`
	warningPublicRegistryFlagIgnoredForPlatformToken = `The feature flag "%s" is set, but it is ignored because a platform token is in use. The public registry endpoint is used automatically with platform tokens.`
)

func publicRegistryOverrideWithoutPublicRegistry(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.PublicRegistryOverride() == "" || dk.FF().IsPublicRegistry() {
		return ""
	}

	// For new DynaKubes (status not yet set), check the token secret directly.
	hasPlatformToken, err := token.NewReader(dv.apiReader, dk).HasPlatformToken(ctx)
	if err == nil && hasPlatformToken {
		return ""
	}

	return fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)
}

func publicRegistryFlagIgnoredForPlatformToken(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if _, hasAnnotation := dk.Annotations[exp.UsePublicRegistryKey]; !hasAnnotation {
		return ""
	}

	hasPlatformToken, err := token.NewReader(dv.apiReader, dk).HasPlatformToken(ctx)
	if err != nil || !hasPlatformToken {
		return ""
	}

	return fmt.Sprintf(warningPublicRegistryFlagIgnoredForPlatformToken, exp.UsePublicRegistryKey)
}
