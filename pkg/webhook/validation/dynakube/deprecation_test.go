package dynakube

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta2.AnnotationFeatureAutomaticInjection: "true",
		}
		dynakube := &dynatracev1beta2.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 1, dynakube)
		assert.True(t, dynakube.FeatureAutomaticInjection())
	})
}

func testDeprecatedAnnotation(t *testing.T,
	newAnnotation string, oldAnnotation string, validatorFunc validator) {
	dynakube := dynatracev1beta2.DynaKube{
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
	t.Run(dynatracev1beta2.AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynatracev1beta2.AnnotationFeatureActiveGateUpdates,
			dynatracev1beta2.AnnotationFeatureDisableActiveGateUpdates,
			deprecatedFeatureFlagDisableActiveGateUpdates)
	})
}

func TestCreateDeprecatedAnnotationWarning(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", dynatracev1beta2.AnnotationFeatureDisableActiveGateUpdates, dynatracev1beta2.AnnotationFeatureActiveGateUpdates),
		deprecatedAnnotationWarning(dynatracev1beta2.AnnotationFeatureActiveGateUpdates, dynatracev1beta2.AnnotationFeatureDisableActiveGateUpdates))
}

func Test_deprecatedFeatureFlagMovedCRDField(t *testing.T) {
	dynakube := dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{},
	}

	dynakube.Annotations = map[string]string{
		dynatracev1beta2.AnnotationFeatureAutomaticInjection: "true",
	}
	assert.Contains(t,
		deprecatedFeatureFlagMovedCRDField(context.Background(), nil, &dynakube),
		"These feature flags are deprecated and will be moved to the CRD in the future",
	)
}
