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

	warningGlobalResourceAttributesExceedsLimit   = "TODO: global resource attributes exceed limit"
	warningOneAgentResourceAttributesExceedsLimit = "TODO: oneAgent resource attributes exceed limit"
	warningOTLPResourceAttributesExceedsLimit     = "TODO: otlpExporterConfiguration resource attributes exceed limit"

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

	return strings.Join(errs, ", ")
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
	if len(dk.Spec.ResourceAttributes) > resourceAttributesLimit {
		return warningGlobalResourceAttributesExceedsLimit
	}

	return ""
}

func oneAgentResourceAttributesExceedsLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if !oa.HasAdditionalResourceAttributes() {
		return ""
	}

	if len(oa.GetResourceAttributes()) > resourceAttributesLimit {
		return warningOneAgentResourceAttributesExceedsLimit
	}

	return ""
}

func otlpResourceAttributesExceedsLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	otlpConfig := dk.OTLPExporterConfiguration()
	if !otlpConfig.HasAdditionalResourceAttributes() {
		return ""
	}

	if len(otlpConfig.GetResourceAttributes()) > resourceAttributesLimit {
		return warningOTLPResourceAttributesExceedsLimit
	}

	return ""
}
