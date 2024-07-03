package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
)

func getDeprecatedFeatureFlagsWillBeRemoved() []string {
	return []string{
		dynakube.AnnotationInjectionFailurePolicy,
	}
}

func getDeprecatedFeatureFlagsWillBeMovedCRD() []string {
	return []string{
		dynakube.AnnotationFeatureAutomaticInjection,
		dynakube.AnnotationFeatureAutomaticK8sApiMonitoring,
		dynakube.AnnotationFeatureActiveGateUpdates,
		dynakube.AnnotationFeatureLabelVersionDetection,
	}
}

func deprecatedFeatureFlagDisableActiveGateUpdates(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	return warnIfDeprecatedIsUsed(dk, dynakube.AnnotationFeatureActiveGateUpdates, dynakube.AnnotationFeatureDisableActiveGateUpdates)
}

func deprecatedFeatureFlagWillBeDeleted(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var featureFlags []string

	for _, ff := range getDeprecatedFeatureFlagsWillBeRemoved() {
		if isDeprecatedFeatureFlagUsed(dk, ff) {
			featureFlags = append(featureFlags, fmt.Sprintf("'%s'", ff))
		}
	}

	if len(featureFlags) == 0 {
		return ""
	}

	return "Some feature flags are deprecated and will be removed in the future: " + strings.Join(featureFlags, ", ")
}

func deprecatedFeatureFlagMovedCRDField(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var featureFlags []string

	for _, ff := range getDeprecatedFeatureFlagsWillBeMovedCRD() {
		if isDeprecatedFeatureFlagUsed(dk, ff) {
			featureFlags = append(featureFlags, fmt.Sprintf("'%s'", ff))
		}
	}

	if len(featureFlags) == 0 {
		return ""
	}

	return "These feature flags are deprecated and will be moved to the CRD in the future: " + strings.Join(featureFlags, ", ")
}

func isDeprecatedFeatureFlagUsed(dk *dynakube.DynaKube, annotation string) bool {
	_, ok := dk.Annotations[annotation]

	return ok
}

func warnIfDeprecatedIsUsed(dk *dynakube.DynaKube, newAnnotation string, deprecatedAnnotation string) string {
	_, hasDeprecatedFlag := dk.Annotations[deprecatedAnnotation]
	if hasDeprecatedFlag {
		return deprecatedAnnotationWarning(newAnnotation, deprecatedAnnotation)
	}

	return ""
}

func deprecatedAnnotationWarning(newAnnotation string, deprecatedAnnotation string) string {
	return fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", deprecatedAnnotation, newAnnotation)
}
