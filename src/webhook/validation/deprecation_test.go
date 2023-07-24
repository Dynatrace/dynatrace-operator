package validation

import (
	"fmt"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy: "false",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 1, dynakube)
		assert.True(t, dynakube.FeatureDisableWebhookReinvocationPolicy())
	})

	t.Run(`warning present`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		split := strings.Split(dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy, "/")
		postFix := split[1]
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.DeprecatedFeatureFlagPrefix + postFix: "true",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 1, dynakube)
		assert.True(t, dynakube.FeatureDisableWebhookReinvocationPolicy())
	})
}

func testDeprecatedAnnotation(t *testing.T,
	newAnnotation string, oldAnnotation string, validatorFunc validator) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{},
	}

	dynakube.Annotations = map[string]string{
		oldAnnotation: "",
	}

	assert.Equal(t,
		deprecatedAnnotationWarning(newAnnotation, oldAnnotation),
		validatorFunc(nil, &dynakube))

	dynakube.Annotations = map[string]string{}

	assert.Empty(t,
		validatorFunc(nil, &dynakube))
}

func TestDeprecatedAnnotationWarnings(t *testing.T) {
	t.Run(dynatracev1beta1.AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureActiveGateUpdates,
			dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates,
			deprecatedFeatureFlagDisableActiveGateUpdates)
	})
	t.Run(dynatracev1beta1.AnnotationFeatureActiveGateRawImage, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureActiveGateRawImage,
			dynatracev1beta1.AnnotationFeatureDisableActiveGateRawImage,
			deprecatedFeatureFlagDisableActiveGateRawImage)
	})
	t.Run(dynatracev1beta1.AnnotationFeatureHostsRequests, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureHostsRequests,
			dynatracev1beta1.AnnotationFeatureDisableHostsRequests,
			deprecatedFeatureFlagDisableHostsRequests)
	})
	t.Run(dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			deprecatedFeatureFlagDisableReadOnlyAgent)
	})
	t.Run(dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy,
			dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy,
			deprecatedFeatureFlagDisableWebhookReinvocationPolicy)
	})
	t.Run(dynatracev1beta1.AnnotationFeatureMetadataEnrichment, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureMetadataEnrichment,
			dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment,
			deprecatedFeatureFlagDisableMetadataEnrichment)
	})
}

func TestCreateDeprecatedAnnotationWarning(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates, dynatracev1beta1.AnnotationFeatureActiveGateUpdates),
		deprecatedAnnotationWarning(dynatracev1beta1.AnnotationFeatureActiveGateUpdates, dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates))

	assert.Equal(t, fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment, dynatracev1beta1.AnnotationFeatureMetadataEnrichment),
		deprecatedAnnotationWarning(dynatracev1beta1.AnnotationFeatureMetadataEnrichment, dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment))
}

func TestDeprecatedFeatureFlagsWarning(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureActiveGateAuthToken),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureActiveGateAuthToken))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureActiveGateReadOnlyFilesystem),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureActiveGateReadOnlyFilesystem))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureActiveGateRawImage),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureActiveGateRawImage))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureActiveGateUpdates),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureActiveGateUpdates))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureHostsRequests),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureHostsRequests))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureWebhookReinvocationPolicy))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureMetadataEnrichment),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureMetadataEnrichment))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureAutomaticInjection),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureAutomaticInjection))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationFeatureLabelVersionDetection),
		deprecationWarning(dynatracev1beta1.AnnotationFeatureLabelVersionDetection))

	assert.Equal(t, fmt.Sprintf("feature flag '%s' is deprecated and will be removed in the future", dynatracev1beta1.AnnotationInjectionFailurePolicy),
		deprecationWarning(dynatracev1beta1.AnnotationInjectionFailurePolicy))
}
