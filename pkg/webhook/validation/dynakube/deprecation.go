package dynakube

import (
	"context"
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
)

func getDeprecatedFeatureFlagsWillBeRemoved() []string {
	return []string{
		dynatracev1beta1.AnnotationInjectionFailurePolicy,
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

func deprecatedFeatureFlagDisableActiveGateUpdates(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnIfDeprecatedIsUsed(dynakube, dynatracev1beta1.AnnotationFeatureActiveGateUpdates, dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates)
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
