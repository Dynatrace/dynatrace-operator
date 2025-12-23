package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestNameStartsWithDigit(t *testing.T) {
	t.Run("dynakube name starts with digit", func(t *testing.T) {
		assertDenied(t, []string{errorNoDNS1053Label}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "1dynakube",
			},
		})
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://tenantid.doma.in/api",
			},
		})
	})
}

func TestNameTooLong(t *testing.T) {
	type testCase struct {
		name            string
		crNameLength    int
		spec            dynakube.DynaKubeSpec
		expectMaxLength int
	}

	testCases := []testCase{
		{"normal length", 10, dynakube.DynaKubeSpec{}, 0},
		{"max - 1", dynakube.MaxNameLength - 1, dynakube.DynaKubeSpec{}, 0},
		{"max", dynakube.MaxNameLength, dynakube.DynaKubeSpec{}, 0},
		{"max + 1", dynakube.MaxNameLength + 1, dynakube.DynaKubeSpec{}, 40},
		{"max oneagent", dynakube.MaxNameLength, dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}}}, 0},
		{"max + 1 oneagent", dynakube.MaxNameLength + 1, dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}}}, 40},
		{"max prometheus", 32, dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}, 0},
		{"max + 1 prometheus", 33, dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}, 32},
		{"max databases", 32, dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "a"}}}}, 0},
		{"max + 1 databases", 33, dynakube.DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "a"}}}}, 32},
		{"max otelc", 38, dynakube.DynaKubeSpec{TelemetryIngest: &telemetryingest.Spec{}}, 0},
		{"max + 1 otelc", 39, dynakube.DynaKubeSpec{TelemetryIngest: &telemetryingest.Spec{}}, 38},
		{"max kspm", 35, dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}, 0},
		{"max + 1 kspm", 36, dynakube.DynaKubeSpec{Kspm: &kspm.Spec{}}, 35},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Repeat("a", test.crNameLength),
				},
				Spec: test.spec,
			}
			dk.Spec.APIURL = "https://tenantid.doma.in/api"
			if dk.Spec.Extensions != nil {
				dk.Spec.Templates.ExtensionExecutionController = extensions.ExecutionControllerSpec{
					ImageRef: image.Ref{Repository: "eec/image", Tag: "latest"},
				}
				dk.Spec.Templates.SQLExtensionExecutor = extensions.DatabaseExecutorSpec{
					ImageRef: image.Ref{Repository: "sqlexec/image", Tag: "latest"},
				}
				dk.Spec.Templates.OpenTelemetryCollector = dynakube.OpenTelemetryCollectorSpec{
					ImageRef: image.Ref{Repository: "otelc/image", Tag: "latest"},
				}
			}
			if dk.Spec.TelemetryIngest != nil {
				dk.Spec.Templates.OpenTelemetryCollector = dynakube.OpenTelemetryCollectorSpec{
					ImageRef: image.Ref{Repository: "otelc/image", Tag: "latest"},
				}
			}
			if dk.Spec.Kspm != nil {
				dk.Spec.Templates.KspmNodeConfigurationCollector = kspm.NodeConfigurationCollectorSpec{
					ImageRef: image.Ref{Repository: "otelc/image", Tag: "latest"},
				}
				dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				}
			}

			if test.expectMaxLength == 0 {
				assertAllowed(t, dk)
			} else {
				msg := fmt.Sprintf(errorNameTooLong, test.expectMaxLength)
				assertDenied(t, []string{msg}, dk)
			}
		})
	}
}

func TestInvalidNameErrorMatches(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo.bar",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenantid.doma.in/api",
		},
	}
	upstreamErr := validation.IsDNS1035Label(dk.Name)
	require.Len(t, upstreamErr, 1)
	assertDenied(t, []string{upstreamErr[0]}, dk)
}

func TestNoNameViolationOnTooLongName(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Repeat("a", 64),
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenantid.doma.in/api",
		},
	}

	_, err := runValidators(dk)
	msg := fmt.Sprintf(errorNameTooLong, dynakube.MaxNameLength)
	require.ErrorContains(t, err, msg)
	assert.NotContains(t, err.Error(), errorNoDNS1053Label)
}
