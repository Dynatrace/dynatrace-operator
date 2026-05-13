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
	// TODO: ICP-3643 - Implement public registry selection based on gen3 platformToken:
	// - If user sets publicRegistryOverride without use-public-registry FF and without platformToken, show error (current behavior)
	// - If user sets use-public-registry FF, ignore it and display warning when platformToken is used
	// - Add logic to allow publicRegistryOverride when platformToken is set, even without use-public-registry FF
	if dk.PublicRegistryOverride() != "" && !dk.FF().IsPublicRegistry() {
		return fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)
	}

	return ""
}
