package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynakube.AnnotationFeatureAutomaticInjection: "true",
		}
		dk := &dynakube.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedWithWarnings(t, 1, dk)
		assert.True(t, dk.FeatureAutomaticInjection())
	})
}

func testDeprecatedAnnotation(t *testing.T,
	newAnnotation string, oldAnnotation string, validatorFunc validatorFunc) {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{},
	}

	dk.Annotations = map[string]string{
		oldAnnotation: "",
	}

	assert.Equal(t,
		deprecatedAnnotationWarning(newAnnotation, oldAnnotation),
		validatorFunc(context.Background(), nil, &dk))

	dk.Annotations = map[string]string{}

	assert.Empty(t,
		validatorFunc(context.Background(), nil, &dk))
}

func TestDeprecatedAnnotationWarnings(t *testing.T) {
	t.Run(dynakube.AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecatedAnnotation(t,
			dynakube.AnnotationFeatureActiveGateUpdates,
			dynakube.AnnotationFeatureDisableActiveGateUpdates,
			deprecatedFeatureFlagDisableActiveGateUpdates)
	})
}

func TestCreateDeprecatedAnnotationWarning(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("annotation '%s' is deprecated, use '%s' instead", dynakube.AnnotationFeatureDisableActiveGateUpdates, dynakube.AnnotationFeatureActiveGateUpdates),
		deprecatedAnnotationWarning(dynakube.AnnotationFeatureActiveGateUpdates, dynakube.AnnotationFeatureDisableActiveGateUpdates))
}

func Test_deprecatedFeatureFlagMovedCRDField(t *testing.T) {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{},
	}

	dk.Annotations = map[string]string{
		dynakube.AnnotationFeatureAutomaticInjection: "true",
	}
	assert.Contains(t,
		deprecatedFeatureFlagMovedCRDField(context.Background(), nil, &dk),
		"These feature flags are deprecated and will be moved to the CRD in the future",
	)
}
