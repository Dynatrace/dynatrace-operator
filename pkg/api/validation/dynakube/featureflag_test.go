package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
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

func TestIsNodeImagePullWithoutCSIDisabled(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		title           string
		csiAvailable    bool
		annotations     map[string]string
		expectedMessage string
	}

	testCases := []testCase{
		{
			title:           "CSI available, node-image-pull not set => no warning",
			csiAvailable:    true,
			annotations:     nil,
			expectedMessage: "",
		},
		{
			title:           "CSI available, node-image-pull enabled => no warning",
			csiAvailable:    true,
			annotations:     map[string]string{exp.OANodeImagePullKey: "true"},
			expectedMessage: "",
		},
		{
			title:           "CSI not available, node-image-pull not set => no warning",
			csiAvailable:    false,
			annotations:     nil,
			expectedMessage: "",
		},
		{
			title:           "CSI not available, node-image-pull explicitly disabled => warning",
			csiAvailable:    false,
			annotations:     map[string]string{exp.OANodeImagePullKey: "false"},
			expectedMessage: warningNodeImagePullWithoutCSI,
		},
		{
			title:           "CSI not available, node-image-pull enabled => warning",
			csiAvailable:    false,
			annotations:     map[string]string{exp.OANodeImagePullKey: "true"},
			expectedMessage: warningNodeImagePullWithoutCSI,
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			installconfig.SetModulesOverride(t, installconfig.Modules{
				CSIDriver:            test.csiAvailable,
				ActiveGate:           true,
				OneAgent:             true,
				Extensions:           true,
				LogMonitoring:        true,
				EdgeConnect:          true,
				Supportability:       true,
				KubernetesMonitoring: true,
				KSPM:                 true,
			})

			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}

			errMsg := isNodeImagePullWithoutCSIDisabled(ctx, &Validator{}, dk)
			assert.Equal(t, test.expectedMessage, errMsg)
		})
	}
}
