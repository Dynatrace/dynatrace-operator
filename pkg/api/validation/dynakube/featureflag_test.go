package validation

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecatedFeatureFlag(t *testing.T) {
	t.Run("Feature flag is deprecated", DeprecatedFeatureFlagWithDeprecatedFlags)
	t.Run("Feature flag is not deprecated", DeprecatedFeatureFlagWithoutDeprecatedFlags)
	t.Run("No annotations", DeprecatedFeatureFlagWithNoAnnotations)
	t.Run("Multiple feature flags are deprecated", DeprecatedFeatureFlagWithMultipleDeprecatedFlags)
}

func DeprecatedFeatureFlagWithDeprecatedFlags(t *testing.T) {
	for _, featureFlag := range deprecatedFeatureFlags {
		t.Run(featureFlag, func(t *testing.T) {
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						featureFlag: "true",
					},
				},
			}
			expected := warningFeatureFlagDeprecated + featureFlag
			result := deprecatedFeatureFlag(t.Context(), nil, dk)

			assert.Equal(t, expected, result)
		})
	}
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
	result := deprecatedFeatureFlag(t.Context(), nil, dk)

	assert.Empty(t, result)
}

func DeprecatedFeatureFlagWithNoAnnotations(t *testing.T) {
	dk := &dynakube.DynaKube{}
	result := deprecatedFeatureFlag(t.Context(), nil, dk)

	assert.Empty(t, result)
}

func DeprecatedFeatureFlagWithMultipleDeprecatedFlags(t *testing.T) {
	annotations := map[string]string{}

	for _, flag := range deprecatedFeatureFlags {
		annotations[flag] = "true"
	}

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Annotations: annotations,
		},
	}

	result := deprecatedFeatureFlag(t.Context(), nil, dk)
	expected := warningFeatureFlagDeprecated + strings.Join(deprecatedFeatureFlags, ", ")
	assert.Equal(t, expected, result)
}
