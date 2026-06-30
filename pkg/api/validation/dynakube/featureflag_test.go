package validation

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecatedFeatureFlag(t *testing.T) {
	t.Run("Feature flag is deprecated", deprecatedFeatureFlagWithDeprecatedFlags)
	t.Run("Feature flag is not deprecated", deprecatedFeatureFlagWithoutDeprecatedFlags)
	t.Run("No annotations", deprecatedFeatureFlagWithNoAnnotations)
	t.Run("Multiple feature flags are deprecated", deprecatedFeatureFlagWithMultipleDeprecatedFlags)
}

func deprecatedFeatureFlagWithDeprecatedFlags(t *testing.T) {
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

func deprecatedFeatureFlagWithoutDeprecatedFlags(t *testing.T) {
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

func deprecatedFeatureFlagWithNoAnnotations(t *testing.T) {
	dk := &dynakube.DynaKube{}
	result := deprecatedFeatureFlag(t.Context(), nil, dk)

	assert.Empty(t, result)
}

func deprecatedFeatureFlagWithMultipleDeprecatedFlags(t *testing.T) {
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

func TestUnknownFeatureFlag(t *testing.T) {
	getDK := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}
	}

	t.Run("no annotations => no warning", func(t *testing.T) {
		dk := getDK()
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("only known flags => no warning", func(t *testing.T) {
		annotations := map[string]string{}
		for _, flag := range knownFeatureFlags {
			if !slices.Contains(deprecatedFeatureFlags, flag) {
				annotations[flag] = "true"
			}
		}

		dk := getDK()
		dk.Annotations = annotations
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("non-feature annotation => no warning", func(t *testing.T) {
		dk := getDK()
		dk.Annotations = map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": "{}",
		}

		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("single unknown feature flag => warning", func(t *testing.T) {
		unknown := exp.FFPrefix + "removed-flag"
		dk := getDK()
		dk.Annotations = map[string]string{unknown: "true"}

		assertAllowedWithWarnings(t, 1, dk)
	})

	t.Run("multiple unknown feature flags => warning with sorted names", func(t *testing.T) {
		unknownA := exp.FFPrefix + "alpha-removed"
		unknownB := exp.FFPrefix + "zeta-removed"

		dk := getDK()
		dk.Annotations = map[string]string{
			unknownB: "true",
			unknownA: "true",
		}

		warnings, err := assertAllowed(t, dk)
		require.NoError(t, err)
		assert.Len(t, warnings, 1)
		assert.Equal(t, fmt.Sprintf(warningFeatureFlagUnknown, unknownA+", "+unknownB), warnings[0])
	})
}

func TestIsNodeImagePullWithoutCSI(t *testing.T) {
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

			errMsg := isNodeImagePullWithoutCSI(t.Context(), &Validator{}, dk)
			assert.Equal(t, test.expectedMessage, errMsg)
		})
	}
}

func TestInvalidNoProxy(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Annotations: map[string]string{}},
		Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
	}

	assertSanitizeArg(t, dk, func(dk *dynakube.DynaKube, value string) {
		dk.Annotations[exp.NoProxyKey] = value
	}, errorInvalidNoProxy)
}
