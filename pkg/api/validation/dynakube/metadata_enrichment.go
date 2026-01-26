package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningMetadataEnrichmentDisabledForInjection = "metadataEnrichment.enabled is set to false, but OneAgent injection is enabled (applicationMonitoring/cloudNativeFullstack). Metadata enrichment will still be applied and this setting is ignored."
)

func disabledMetadataEnrichmentForInjectionModes(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if dk.Spec.MetadataEnrichment.Enabled == nil || *dk.Spec.MetadataEnrichment.Enabled {
		return ""
	}

	if dk.OneAgent().IsAppInjectionNeeded() {
		return warningMetadataEnrichmentDisabledForInjection
	}

	return ""
}
