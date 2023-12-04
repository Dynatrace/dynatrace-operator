package dynakube

import (
	"context"
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
)

const (
	featureDeprecatedWarningMessage = `DEPRECATED: %s`
)

func getDeprecatedFeatureFlagsWillBeRemoved() []string {
	return []string{
		dynatracev1beta1.AnnotationInjectionFailurePolicy,
		dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy,
		dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
		dynatracev1beta1.AnnotationFeatureHostsRequests,
		dynatracev1beta1.AnnotationFeatureActiveGateAuthToken,
		dynatracev1beta1.AnnotationFeatureActiveGateRawImage,
		dynatracev1beta1.AnnotationFeatureActiveGateReadOnlyFilesystem,
	}
}

func getDeprecatedFeatureFlagsWillBeMovedCRD() []string {
	return []string{
		dynatracev1beta1.AnnotationFeatureAutomaticInjection,
		dynatracev1beta1.AnnotationFeatureMetadataEnrichment,
		dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring,
		dynatracev1beta1.AnnotationFeatureActiveGateUpdates,
		dynatracev1beta1.AnnotationFeatureLabelVersionDetection,
	}
}

func deprecatedFeatureFlagFormat(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Annotations == nil {
		return ""
	}

	deprecatedPrefix := dynatracev1beta1.DeprecatedFeatureFlagPrefix
	if len(dynatracev1beta1.FlagsWithPrefix(dynakube, deprecatedPrefix)) > 0 {
		return fmt.Sprintf(featureDeprecatedWarningMessage, "'alpha.operator.dynatrace.com/feature-' prefix will be replaced with the 'feature.dynatrace.com/' prefix for dynakube feature-flags")
	}

	return ""
}

func deprecatedFeatureFlagDisableActiveGateUpdates(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureActiveGateUpdates, dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates)
}

func deprecatedFeatureFlagDisableActiveGateRawImage(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureActiveGateRawImage, dynatracev1beta1.AnnotationFeatureDisableActiveGateRawImage)
}

func deprecatedFeatureFlagDisableHostsRequests(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureHostsRequests, dynatracev1beta1.AnnotationFeatureDisableHostsRequests)
}

func deprecatedFeatureFlagDisableReadOnlyAgent(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent, dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent)
}

func deprecatedFeatureFlagDisableWebhookReinvocationPolicy(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy, dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy)
}

func deprecatedFeatureFlagDisableMetadataEnrichment(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureMetadataEnrichment, dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment)
}

func deprecatedFeatureFlagWillBeDeleted(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	var featureFlags []string
	for _, ff := range getDeprecatedFeatureFlagsWillBeRemoved() {
		if isDeprecatedFeatureFlagUsed(dynakube, ff) {
			featureFlags = append(featureFlags, fmt.Sprintf("'%s'", ff))
		}
	}

	if len(featureFlags) == 0 {
		return ""
	}
	return "Some feature flags are deprecated and will be removed in the future: " + strings.Join(featureFlags, ", ")
}

func deprecatedFeatureFlagMovedCRDField(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	var featureFlags []string
	for _, ff := range getDeprecatedFeatureFlagsWillBeMovedCRD() {
		if isDeprecatedFeatureFlagUsed(dynakube, ff) {
			featureFlags = append(featureFlags, fmt.Sprintf("'%s'", ff))
		}
	}

	if len(featureFlags) == 0 {
		return ""
	}
	return "These feature flags are deprecated and will be moved to the CRD in the future: " + strings.Join(featureFlags, ", ")
}

func isDeprecatedFeatureFlagUsed(dynakube *dynatracev1beta1.DynaKube, annotation string) bool {
	_, ok := dynakube.Annotations[annotation]
	return ok
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
