package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDynakubeName = "dynakube"

func TestExtensionsWithoutK8SMonitoring(t *testing.T) {
	runExtensionTestCases(t,
		extensionTestCase{
			"no error if activegate with k8s-monitoring",
			func(t *testing.T, setExtensions dkMutatorFunc) {
				dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
				dk.Spec.ActiveGate = activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				}
				assertAllowed(t, setExtensions(dk))
			},
		},

		extensionTestCase{
			"error if no activegate with k8s-monitoring",
			func(t *testing.T, setExtensions dkMutatorFunc) {
				assertAllowedWithWarnings(t, 2, setExtensions(createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)))
			},
		},

		extensionTestCase{
			"error if activegate with k8s-monitoring but automatic Kuberenetes API monitoring is disabled",
			func(t *testing.T, setExtensions dkMutatorFunc) {
				dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
				dk.Annotations = map[string]string{
					exp.AGAutomaticK8sAPIMonitoringKey: "false",
				}
				dk.Spec.ActiveGate = activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				}
				assertAllowedWithWarnings(t, 2, setExtensions(dk))
			},
		},

		extensionTestCase{
			"error if automatic Kuberenetes API monitoring is disabled and without activgate k8s-monitoring",
			func(t *testing.T, setExtensions dkMutatorFunc) {
				dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
				dk.Annotations = map[string]string{
					exp.AGAutomaticK8sAPIMonitoringKey: "false",
				}
				assertAllowedWithWarnings(t, 2, setExtensions(dk))
			},
		},
	)
}

func TestExtensionsWithoutOtelcImage(t *testing.T) {
	t.Run("error when image is not specified", func(t *testing.T) {
		assertDenied(t, []string{errorOtelCollectorMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testAPIURL,
					Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
				},
			})
	})
}

func createStandaloneExtensionsDynakube(name, apiURL string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiURL,
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: "repo/image",
						Tag:        "version",
					},
				},
				OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
					ImageRef: image.Ref{
						Repository: "repo/otel-collector",
						Tag:        "version",
					},
				},
				DatabaseExecutor: extensions.DatabaseExecutorSpec{
					ImageRef: image.Ref{
						Repository: "repo/image",
						Tag:        "version",
					},
				},
			},
		},
	}

	return dk
}

type extensionTestCase struct {
	title string
	test  func(t *testing.T, setExtensions dkMutatorFunc)
}

type dkMutatorFunc func(*dynakube.DynaKube) *dynakube.DynaKube

func runExtensionTestCases(t *testing.T, cases ...extensionTestCase) {
	matrix := []struct {
		name string
		spec *extensions.Spec
	}{
		{"prometheus extension enabled: ", &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}},
		{"databases extension enabled:", &extensions.Spec{Databases: []extensions.DatabaseSpec{{ID: "test"}}}},
		{"all extensions enabled:", &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}, Databases: []extensions.DatabaseSpec{{ID: "test"}}}},
	}

	for _, tt := range matrix {
		for _, tc := range cases {
			name := tt.name + ":" + tc.title
			t.Run(name, func(t *testing.T) {
				tc.test(t, func(dk *dynakube.DynaKube) *dynakube.DynaKube {
					dk.Spec.Extensions = tt.spec

					return dk
				})
			})
		}
	}
}
