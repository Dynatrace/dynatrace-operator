package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorPublicRegistryOverrideWithoutPublicRegistry = `The publicRegistryOverride field is set, but the feature flag "%s" is not enabled. Either enable the feature flag or remove the publicRegistryOverride field.`
)

func publicRegistryOverrideWithoutPublicRegistry(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.PublicRegistryOverride() != "" && !dk.FF().IsPublicRegistry() {
		return fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)
	}

	return ""
}
