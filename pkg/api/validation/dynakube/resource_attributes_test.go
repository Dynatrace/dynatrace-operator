package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeStringMapWithPrefix creates a map with n unique entries using keys prefixed by the given prefix.
func makeStringMapWithPrefix(prefix string, n int) map[string]string {
	m := make(map[string]string, n)
	for i := range n {
		key := fmt.Sprintf("%s%d", prefix, i)
		m[key] = key
	}

	return m
}

func TestResourceAttributesValidation(t *testing.T) {
	t.Run("no warning when all counts are within limit", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 3),
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							// 2 additional non-overlapping keys → merged = 5 (≤ limit)
							AdditionalResourceAttributes: makeStringMapWithPrefix("a", 2),
						},
					},
				},
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					// 2 additional non-overlapping keys → merged = 5 (≤ limit)
					AdditionalResourceAttributes: makeStringMapWithPrefix("o", 2),
				},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("global warning fires when global count exceeds limit", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 6),
			},
		}
		// global>5 → globalResourceAttributesExceedsLimit fires;
		// no component additionalResourceAttributes configured → component validators do not fire
		warnings, _ := assertAllowed(t, dk)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings, warningGlobalResourceAttributesExceedsLimit)
	})

	t.Run("no global warning when global count equals limit", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 5),
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("component warning when global within limit but component merged count exceeds limit", func(t *testing.T) {
		// global: 3 keys (g0,g1,g2); additional: 3 distinct keys (a0,a1,a2) → merged = 6 > 5
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 3),
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							AdditionalResourceAttributes: makeStringMapWithPrefix("a", 3),
						},
					},
				},
			},
		}
		warnings, _ := assertAllowed(t, dk)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings, warningOneAgentResourceAttributesExceedsLimit)
	})

	t.Run("both global and component warning when both exceed limit", func(t *testing.T) {
		// global: 6 keys → global warning;
		// oneAgent additional: 1 non-overlapping key → merged = 7 > 5 → oneAgent warning;
		// no OTLP additionalResourceAttributes → OTLP validator does not fire
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 6),
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							AdditionalResourceAttributes: makeStringMapWithPrefix("a", 1),
						},
					},
				},
			},
		}
		warnings, _ := assertAllowed(t, dk)
		assert.Len(t, warnings, 2)
		assert.Contains(t, warnings, warningGlobalResourceAttributesExceedsLimit)
		assert.Contains(t, warnings, warningOneAgentResourceAttributesExceedsLimit)
	})

	t.Run("multiple components each exceeding threshold independently emit one warning each", func(t *testing.T) {
		// global: 3 (g0..g2); oneAgent additional: 3 (a0..a2) → merged=6; otlp additional: 3 (o0..o2) → merged=6
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 3),
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							AdditionalResourceAttributes: makeStringMapWithPrefix("a", 3),
						},
					},
				},
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					AdditionalResourceAttributes: makeStringMapWithPrefix("o", 3),
				},
			},
		}
		warnings, _ := assertAllowed(t, dk)
		assert.Len(t, warnings, 2)
		assert.Contains(t, warnings, warningOneAgentResourceAttributesExceedsLimit)
		assert.Contains(t, warnings, warningOTLPResourceAttributesExceedsLimit)
	})

	t.Run("overlapping keys between global and additional dedup keeps merged count within limit", func(t *testing.T) {
		// 3 global keys + 3 additional where 2 overlap (g0, g1 shared; a0 unique) → merged = 4 (≤ limit)
		global := map[string]string{"g0": "global", "g1": "global", "g2": "global"}
		additional := map[string]string{"g0": "additional", "g1": "additional", "a0": "additional"}

		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: global,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							AdditionalResourceAttributes: additional,
						},
					},
				},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("OTLP warning only when OTLP component merged count exceeds limit", func(t *testing.T) {
		// global: 3 (g0..g2); otlp additional: 3 (o0..o2) → merged = 6 > 5
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: makeStringMapWithPrefix("g", 3),
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					AdditionalResourceAttributes: makeStringMapWithPrefix("o", 3),
				},
			},
		}
		warnings, _ := assertAllowed(t, dk)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings, warningOTLPResourceAttributesExceedsLimit)
	})
}

func TestValidateResourceAttributeMap(t *testing.T) {
	tests := []struct {
		name        string
		attrs       map[string]string
		expectEmpty bool
	}{
		{
			name:        "nil map is valid",
			attrs:       nil,
			expectEmpty: true,
		},
		{
			name:        "empty map is valid",
			attrs:       map[string]string{},
			expectEmpty: true,
		},
		{
			name:        "simple key and value are valid",
			attrs:       map[string]string{"team": "platform"},
			expectEmpty: true,
		},
		{
			name:        "qualified key with prefix is valid",
			attrs:       map[string]string{"app.kubernetes.io/name": "my-app"},
			expectEmpty: true,
		},
		{
			name:        "key with spaces is invalid",
			attrs:       map[string]string{"my key": "value"},
			expectEmpty: false,
		},
		{
			name:        "key with special characters is invalid",
			attrs:       map[string]string{"my!key": "value"},
			expectEmpty: false,
		},
		{
			name:        "value with spaces is invalid (label value constraint)",
			attrs:       map[string]string{"team": "my platform"},
			expectEmpty: false,
		},
		{
			name:        "value longer than 63 chars is invalid",
			attrs:       map[string]string{"team": strings.Repeat("a", 64)},
			expectEmpty: false,
		},
		{
			name:        "multiple violations are all reported",
			attrs:       map[string]string{"Bad Key": "bad value!"},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateResourceAttributeMap(tt.attrs)
			if tt.expectEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestResourceAttributesSyntaxValidation(t *testing.T) {
	t.Run("valid global attributes are accepted", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: map[string]string{"team": "platform", "env": "dev"},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("invalid global attribute key is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: map[string]string{"Invalid Key": "value"},
			},
		}
		assertDenied(t, []string{"spec.resourceAttributes contains invalid entries"}, dk)
	})

	t.Run("invalid global attribute value is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: map[string]string{"team": "my platform"},
			},
		}
		assertDenied(t, []string{"spec.resourceAttributes contains invalid entries"}, dk)
	})

	t.Run("invalid oneAgent additionalResourceAttributes key is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							AdditionalResourceAttributes: map[string]string{"Invalid Key": "value"},
						},
					},
				},
			},
		}
		assertDenied(t, []string{"spec.oneAgent.*.additionalResourceAttributes contains invalid entries"}, dk)
	})

	t.Run("invalid OTLP additionalResourceAttributes value is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					AdditionalResourceAttributes: map[string]string{"team": "my platform"},
				},
			},
		}
		assertDenied(t, []string{"spec.otlpExporterConfiguration.additionalResourceAttributes contains invalid entries"}, dk)
	})

	t.Run("no component additional attributes configured - invalid global does not bleed into component error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL:             testAPIURL,
				ResourceAttributes: map[string]string{"Invalid Key": "value"},
			},
		}
		_, err := runValidators(t, dk)
		require.ErrorContains(t, err, "spec.resourceAttributes contains invalid entries")
		assert.NotContains(t, err.Error(), "spec.oneAgent")
		assert.NotContains(t, err.Error(), "spec.otlpExporterConfiguration")
	})
}
