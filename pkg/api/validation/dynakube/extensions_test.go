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
	t.Run("no error if extensions are enabled with activegate with k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowed(t, dk)
	})
	t.Run("error if extensions are enabled without activegate with k8s-monitoring", func(t *testing.T) {
		assertAllowedWithWarnings(t, 2,
			createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL))
	})
	t.Run("error if extensions are enabled with activegate with k8s-monitoring but automatic Kuberenetes API monitoring is disabled", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Annotations = map[string]string{
			exp.AGAutomaticK8sAPIMonitoringKey: "false",
		}
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowedWithWarnings(t, 2, dk)
	})
	t.Run("error if extensions are enabled but automatic Kuberenetes API monitoring is disabled and without activgate k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testAPIURL)
		dk.Annotations = map[string]string{
			exp.AGAutomaticK8sAPIMonitoringKey: "false",
		}
		assertAllowedWithWarnings(t, 2, dk)
	})
}

func createStandaloneExtensionsDynakube(name, apiURL string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     apiURL,
			Extensions: &extensions.Spec{PrometheusSpec: &extensions.PrometheusSpec{}},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
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
