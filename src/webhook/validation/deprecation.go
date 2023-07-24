package validation

import (
	"fmt"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	featureDeprecatedWarningMessage = `DEPRECATED: %s`
)

func getDeprecatedFeatureFlags() []string {
	return []string{
		dynatracev1beta1.AnnotationInjectionFailurePolicy,
		dynatracev1beta1.AnnotationFeatureLabelVersionDetection,
		dynatracev1beta1.AnnotationFeatureAutomaticInjection,
		dynatracev1beta1.AnnotationFeatureMetadataEnrichment,
		dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy,
		dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
		dynatracev1beta1.AnnotationFeatureHostsRequests,
		dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring,
		dynatracev1beta1.AnnotationFeatureActiveGateUpdates,
		dynatracev1beta1.AnnotationFeatureActiveGateAuthToken,
		dynatracev1beta1.AnnotationFeatureActiveGateRawImage,
		dynatracev1beta1.AnnotationFeatureActiveGateReadOnlyFilesystem,
	}
}

func deprecatedFeatureFlagFormat(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Annotations == nil {
		return ""
	}

	deprecatedPrefix := dynatracev1beta1.DeprecatedFeatureFlagPrefix
	if len(dynatracev1beta1.FlagsWithPrefix(dynakube, deprecatedPrefix)) > 0 {
		return fmt.Sprintf(featureDeprecatedWarningMessage, "'alpha.operator.dynatrace.com/feature-' prefix will be replaced with the 'feature.dynatrace.com/' prefix for dynakube feature-flags")
	}

	return ""
}

func deprecatedFeatureFlagDisableActiveGateUpdates(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureActiveGateUpdates, dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates)
}

func deprecatedFeatureFlagDisableActiveGateRawImage(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureActiveGateRawImage, dynatracev1beta1.AnnotationFeatureDisableActiveGateRawImage)
}

func deprecatedFeatureFlagDisableHostsRequests(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureHostsRequests, dynatracev1beta1.AnnotationFeatureDisableHostsRequests)
}

func deprecatedFeatureFlagDisableReadOnlyAgent(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent, dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent)
}

func deprecatedFeatureFlagDisableWebhookReinvocationPolicy(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy, dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy)
}

func deprecatedFeatureFlagDisableMetadataEnrichment(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureMetadataEnrichment, dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment)
}

func deprecatedFeatureFlag(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	warning := ""
	for _, ff := range getDeprecatedFeatureFlags() {
		warning += checkDeprecatedFeatureFlag(dynakube, ff)

	}
	return warning
}

func warnIfDeprecatedIsUsed(dynakube *dynatracev1beta1.DynaKube, newAnnotation string, deprecatedAnnotation string) string {
	_, hasDeprecatedFlag := dynakube.Annotations[deprecatedAnnotation]
	if hasDeprecatedFlag {
		return deprecatedAnnotationWarning(newAnnotation, deprecatedAnnotation)
	}

	return ""
}

func deprecatedAnnotationWarning(newAnnotation string, deprecatedAnnotation string) string {
	return fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", deprecatedAnnotation, newAnnotation)
}

func checkDeprecatedFeatureFlag(dynakube *dynatracev1beta1.DynaKube, annotation string) string {
	if _, ok := dynakube.Annotations[annotation]; ok {
		return deprecationWarning(annotation)
	}

	return ""
}

func deprecationWarning(annotation string) string {
	return fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future\n", annotation)
}
