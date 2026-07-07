package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/utils/ptr"
)

const (
	errorPublicRegistryOverrideWithoutPublicRegistry    = `The publicRegistryOverride field is set, but the feature flag "%s" is not enabled. Either enable the feature flag or remove the publicRegistryOverride field.`
	warningPublicRegistryFlagIgnoredForPlatformToken    = `The feature flag "%s" is set, but it is ignored because a platform token is in use. The public registry endpoint is used automatically with platform tokens.`
	errorClassicFullStackIncompatibleWithPublicRegistry = `The DynaKube's specification uses classicFullStack, which is not compatible with the public registry feature or platform tokens. Consider upgrading to cloudNativeFullStack.`
)

func publicRegistryOverrideWithoutPublicRegistry(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.PublicRegistryOverride() == "" || dk.FF().IsPublicRegistry() {
		return ""
	}

	return fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)
}

func publicRegistryFlagIgnoredForPlatformToken(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if _, hasAnnotation := dk.Annotations[exp.UsePublicRegistryKey]; !hasAnnotation {
		return ""
	}

	if !ptr.Deref(dk.Status.APIToken.Platform, false) {
		return ""
	}

	return fmt.Sprintf(warningPublicRegistryFlagIgnoredForPlatformToken, exp.UsePublicRegistryKey)
}

func publicRegistryNotAllowedForClassic(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsClassicFullStackMode() {
		return ""
	}

	if dk.PublicRegistryOverride() != "" || dk.FF().IsPublicRegistry() {
		return errorClassicFullStackIncompatibleWithPublicRegistry
	}

	return ""
}
