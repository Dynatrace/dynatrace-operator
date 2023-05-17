package validation

import (
	"fmt"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
)

const (
	featureDeprecatedWarningMessage = `DEPRECATED: %s`
)

func deprecatedFeatureFlagFormat(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	if dynakube.Annotations == nil {
		return ""
	}

	deprecatedPrefix := dynatracev1.DeprecatedFeatureFlagPrefix
	if len(dynatracev1.FlagsWithPrefix(dynakube, deprecatedPrefix)) > 0 {
		return fmt.Sprintf(featureDeprecatedWarningMessage, "'alpha.operator.dynatrace.com/feature-' prefix will be replaced with the 'feature.dynatrace.com/' prefix for dynakube feature-flags")
	}

	return ""
}

func deprecatedFeatureFlagDisableActiveGateUpdates(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureActiveGateUpdates, dynatracev1.AnnotationFeatureDisableActiveGateUpdates)
}

func deprecatedFeatureFlagDisableActiveGateRawImage(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureActiveGateRawImage, dynatracev1.AnnotationFeatureDisableActiveGateRawImage)
}

func deprecatedFeatureFlagDisableHostsRequests(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureHostsRequests, dynatracev1.AnnotationFeatureDisableHostsRequests)
}

func deprecatedFeatureFlagDisableReadOnlyAgent(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureReadOnlyOneAgent, dynatracev1.AnnotationFeatureDisableReadOnlyOneAgent)
}

func deprecatedFeatureFlagDisableWebhookReinvocationPolicy(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureWebhookReinvocationPolicy, dynatracev1.AnnotationFeatureDisableWebhookReinvocationPolicy)
}

func deprecatedFeatureFlagDisableMetadataEnrichment(_ *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1.AnnotationFeatureMetadataEnrichment, dynatracev1.AnnotationFeatureDisableMetadataEnrichment)
}

func warnIfDeprecatedIsUsed(dynakube *dynatracev1.DynaKube, newAnnotation string, deprecatedAnnotation string) string {
	_, hasDeprecatedFlag := dynakube.Annotations[deprecatedAnnotation]
	if hasDeprecatedFlag {
		return deprecatedAnnotationWarning(newAnnotation, deprecatedAnnotation)
	}

	return ""
}

func deprecatedAnnotationWarning(newAnnotation string, deprecatedAnnotation string) string {
	return fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", deprecatedAnnotation, newAnnotation)
}
