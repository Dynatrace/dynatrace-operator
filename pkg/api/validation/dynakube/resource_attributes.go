package validation

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/resourceattributes"
	"k8s.io/apimachinery/pkg/api/validate/content"
)

const (
	resourceAttributesLimit = 10

	warningGlobalResourceAttributesExceedsLimit   = "This resource defines %d global resource attributes, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."
	warningOneAgentResourceAttributesExceedsLimit = "This resource defines %d resource attributes for the OneAgent, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."
	warningOTLPResourceAttributesExceedsLimit     = "This resource defines %d resource attributes for OTLP exporter auto-configuration, which exceeds the recommended limit of 10. Attributes increase ingestion volume resulting in higher ingest cost. Consider removing attributes or consolidating metadata before applying this DynaKube resource."

	errorResourceAttributesInvalidGlobal   = "spec.resourceAttributes contains invalid entries: %s"
	errorResourceAttributesInvalidOneAgent = "spec.oneAgent.*.additionalResourceAttributes contains invalid entries: %s"
	errorResourceAttributesInvalidOTLP     = "spec.otlpExporterConfiguration.additionalResourceAttributes contains invalid entries: %s"

	warnResourceAttributesSanitizationGlobal   = "spec.resourceAttributes contains invalid keys that will be automatically renamed:%s\nConsider updating these keys in your DynaKube spec to avoid confusion."
	warnResourceAttributesSanitizationOneAgent = "spec.oneAgent.*.additionalResourceAttributes contains invalid keys that will be automatically renamed:%s\nConsider updating these keys in your DynaKube spec to avoid confusion."
	warnResourceAttributesSanitizationOTLP     = "spec.otlpExporterConfiguration.additionalResourceAttributes contains invalid keys that will be automatically renamed:%s\nConsider updating these keys in your DynaKube spec to avoid confusion."

	errorResourceAttributesSanitizationGlobal   = "spec.resourceAttributes contains invalid keys:%s"
	errorResourceAttributesSanitizationOneAgent = "spec.oneAgent.*.additionalResourceAttributes contains invalid keys:%s"
	errorResourceAttributesSanitizationOTLP     = "spec.otlpExporterConfiguration.additionalResourceAttributes contains invalid keys:%s"
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

// checkResourceAttributeSanitization returns warning and error descriptions for keys that would
// be renamed or dropped when sanitized for use as Kubernetes annotation name segments.
// Warning: a key contains characters that will be replaced (renamed but non-empty result).
// Error: a key sanitizes to an empty string (will be dropped) or two keys produce the same
// sanitized value (ambiguous collision).
func checkResourceAttributeSanitization(attrs map[string]string) (warns, errs string) {
	type entry struct {
		original  string
		sanitized string
	}

	entries := make([]entry, 0, len(attrs))
	for key := range attrs {
		entries = append(entries, entry{original: key, sanitized: resourceattributes.SanitizeKey(key)})
	}

	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.original, b.original) })

	var warnParts, errParts []string

	seen := map[string]string{} // sanitized → first original that produced it

	for _, e := range entries {
		if e.sanitized == "" {
			errParts = append(errParts, fmt.Sprintf("\n    * key %q will be dropped — no valid characters remain after sanitization", e.original))

			continue
		}

		if existing, collision := seen[e.sanitized]; collision {
			errParts = append(errParts, fmt.Sprintf("\n    * keys %q and %q both sanitize to %q — rename one to avoid an ambiguous collision", existing, e.original, e.sanitized))
		} else {
			seen[e.sanitized] = e.original
		}

		if e.sanitized != e.original {
			warnParts = append(warnParts, fmt.Sprintf("\n    * key %q will be renamed to %q", e.original, e.sanitized))
		}
	}

	return strings.Join(warnParts, ""), strings.Join(errParts, "")
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

func warnGlobalResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	warns, _ := checkResourceAttributeSanitization(dk.Spec.ResourceAttributes)
	if warns != "" {
		return fmt.Sprintf(warnResourceAttributesSanitizationGlobal, warns)
	}

	return ""
}

func invalidGlobalResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	_, errs := checkResourceAttributeSanitization(dk.Spec.ResourceAttributes)
	if errs != "" {
		return fmt.Sprintf(errorResourceAttributesSanitizationGlobal, errs)
	}

	return ""
}

func warnOneAgentResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if !oa.HasAdditionalResourceAttributes() {
		return ""
	}

	warns, _ := checkResourceAttributeSanitization(oa.GetAdditionalResourceAttributes())
	if warns != "" {
		return fmt.Sprintf(warnResourceAttributesSanitizationOneAgent, warns)
	}

	return ""
}

func invalidOneAgentResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if !oa.HasAdditionalResourceAttributes() {
		return ""
	}

	_, errs := checkResourceAttributeSanitization(oa.GetAdditionalResourceAttributes())
	if errs != "" {
		return fmt.Sprintf(errorResourceAttributesSanitizationOneAgent, errs)
	}

	return ""
}

func warnOTLPResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	otlpConfig := dk.OTLPExporterConfiguration()
	if !otlpConfig.HasAdditionalResourceAttributes() {
		return ""
	}

	warns, _ := checkResourceAttributeSanitization(otlpConfig.GetAdditionalResourceAttributes())
	if warns != "" {
		return fmt.Sprintf(warnResourceAttributesSanitizationOTLP, warns)
	}

	return ""
}

func invalidOTLPResourceAttributesSanitization(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	otlpConfig := dk.OTLPExporterConfiguration()
	if !otlpConfig.HasAdditionalResourceAttributes() {
		return ""
	}

	_, errs := checkResourceAttributeSanitization(otlpConfig.GetAdditionalResourceAttributes())
	if errs != "" {
		return fmt.Sprintf(errorResourceAttributesSanitizationOTLP, errs)
	}

	return ""
}
