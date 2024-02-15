package dynakube

import (
	"context"
	"fmt"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureAutomaticInjection: "true",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 1, dynakube)
		assert.True(t, dynakube.FeatureAutomaticInjection())
	})

	t.Run(`warning not present anymore`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		split := strings.Split(dynatracev1beta1.AnnotationFeatureAutomaticInjection, "/")
		postFix := split[1]
		dynakubeMeta.Annotations = map[string]string{
			`alpha.operator.dynatrace.com/feature-` + postFix: "false",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 0, dynakube)
		assert.True(t, dynakube.FeatureAutomaticInjection())
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
		validatorFunc(context.Background(), nil, &dynakube))

	dynakube.Annotations = map[string]string{}

	assert.Empty(t,
		validatorFunc(context.Background(), nil, &dynakube))
}

func TestDeprecatedAnnotationWarnings(t *testing.T) {
	t.Run(dynatracev1beta1.AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta1.AnnotationFeatureActiveGateUpdates,
			dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates,
			deprecatedFeatureFlagDisableActiveGateUpdates)
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

func Test_deprecatedFeatureFlagMovedCRDField(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{},
	}

	dynakube.Annotations = map[string]string{
		dynatracev1beta1.AnnotationFeatureAutomaticInjection: "true",
	}
	assert.Contains(t,
		deprecatedFeatureFlagMovedCRDField(context.Background(), nil, &dynakube),
		"These feature flags are deprecated and will be moved to the CRD in the future",
	)
}
