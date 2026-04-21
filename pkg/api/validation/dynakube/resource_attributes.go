package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/apimachinery/pkg/api/validate/content"
)

const (
	resourceAttributesLimit = 5

	warningGlobalResourceAttributesExceedsLimit   = "This resource defines %d global resource attributes, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."
	warningOneAgentResourceAttributesExceedsLimit = "This resource defines %d resource attributes for the OneAgent, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."
	warningOTLPResourceAttributesExceedsLimit     = "This resource defines %d resource attributes for OTLP exporter auto-configuration, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."

	errorResourceAttributesInvalidGlobal   = "spec.resourceAttributes contains invalid entries: %s"
	errorResourceAttributesInvalidOneAgent = "spec.oneAgent.*.additionalResourceAttributes contains invalid entries: %s"
	errorResourceAttributesInvalidOTLP     = "spec.otlpExporterConfiguration.additionalResourceAttributes contains invalid entries: %s"
)

// validateResourceAttributeMap checks that every key is a qualified name and every value is a valid label value.
// Returns a comma-separated description of all violations, or an empty string when all entries are valid.
func validateResourceAttributeMap(attrs map[string]string) string {
	var errs []string

	for k, v := range attrs {
		if keyErrs := content.IsLabelKey(k); len(keyErrs) > 0 {
			errs = append(errs, fmt.Sprintf("\n    * invalid key %q: %s", k, strings.Join(keyErrs, "; ")))
		}

		if valErrs := content.IsLabelValue(v); len(valErrs) > 0 {
			errs = append(errs, fmt.Sprintf("\n    * invalid value %q for key %q: %s", v, k, strings.Join(valErrs, "; ")))
		}
	}

	return strings.Join(errs, "")
}

func invalidGlobalResourceAttributes(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if errs := validateResourceAttributeMap(dk.Spec.ResourceAttributes); errs != "" {
		return fmt.Sprintf(errorResourceAttributesInvalidGlobal, errs)
	}

	return ""
}

func invalidOneAgentResourceAttributes(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if !oa.HasAdditionalResourceAttributes() {
		return ""
	}

	if errs := validateResourceAttributeMap(oa.GetAdditionalResourceAttributes()); errs != "" {
		return fmt.Sprintf(errorResourceAttributesInvalidOneAgent, errs)
	}

	return ""
}

func invalidOTLPResourceAttributes(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	otlpConfig := dk.OTLPExporterConfiguration()
	if !otlpConfig.HasAdditionalResourceAttributes() {
		return ""
	}

	if errs := validateResourceAttributeMap(otlpConfig.GetAdditionalResourceAttributes()); errs != "" {
		return fmt.Sprintf(errorResourceAttributesInvalidOTLP, errs)
	}

	return ""
}

func globalResourceAttributesExceedsLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	count := len(dk.Spec.ResourceAttributes)
	if count > resourceAttributesLimit {
		return fmt.Sprintf(warningGlobalResourceAttributesExceedsLimit, count)
	}

	return ""
}

func oneAgentResourceAttributesExceedsLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if !oa.HasAdditionalResourceAttributes() {
		return ""
	}

	count := len(oa.GetResourceAttributes())
	if count > resourceAttributesLimit {
		return fmt.Sprintf(warningOneAgentResourceAttributesExceedsLimit, count)
	}

	return ""
}

func otlpResourceAttributesExceedsLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	otlpConfig := dk.OTLPExporterConfiguration()
	if !otlpConfig.HasAdditionalResourceAttributes() {
		return ""
	}

	count := len(otlpConfig.GetResourceAttributes())
	if count > resourceAttributesLimit {
		return fmt.Sprintf(warningOTLPResourceAttributesExceedsLimit, count)
	}

	return ""
}
