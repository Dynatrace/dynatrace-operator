package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecatedFeatureFlag(t *testing.T) {
	t.Run(`Feature flag is deprecated`, DeprecatedFeatureFlagWithDeprecatedFlags)
	t.Run(`Feature flag is not deprecated`, DeprecatedFeatureFlagWithoutDeprecatedFlags)
	t.Run(`No annotations`, DeprecatedFeatureFlagWithNoAnnotations)
}

func DeprecatedFeatureFlagWithDeprecatedFlags(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				dynakube.AnnotationFeatureOneAgentIgnoreProxy: "true", //nolint:staticcheck
			},
		},
	}
	expected := fmt.Sprintf(warningFeatureFlagDeprecated, dynakube.AnnotationFeatureOneAgentIgnoreProxy) //nolint:staticcheck
	result := deprecatedFeatureFlag(context.Background(), nil, dk)

	assert.Equal(t, expected, result)
}

func DeprecatedFeatureFlagWithoutDeprecatedFlags(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				"other-flag": "true",
			},
		},
	}
	result := deprecatedFeatureFlag(context.Background(), nil, dk)

	assert.Empty(t, result)
}

func DeprecatedFeatureFlagWithNoAnnotations(t *testing.T) {
	dk := &dynakube.DynaKube{}
	result := deprecatedFeatureFlag(context.Background(), nil, dk)

	assert.Empty(t, result)
}
