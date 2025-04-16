package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDynakubeName = "dynakube"

func TestExtensionsWithoutK8SMonitoring(t *testing.T) {
	t.Run("no error if extensions are enabled with activegate with k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testApiUrl)
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowed(t, dk)
	})
	t.Run("error if extensions are enabled without activegate with k8s-monitoring", func(t *testing.T) {
		assertAllowedWithWarnings(t, 2,
			createStandaloneExtensionsDynakube(testDynakubeName, testApiUrl))
	})
	t.Run("error if extensions are enabled with activegate with k8s-monitoring but automatic Kuberenetes API monitoring is disabled", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testApiUrl)
		dk.ObjectMeta.Annotations = map[string]string{
			exp.AGAutomaticK8sApiMonitoringKey: "false",
		}
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
			},
		}
		assertAllowedWithWarnings(t, 2, dk)
	})
	t.Run("error if extensions are enabled but automatic Kuberenetes API monitoring is disabled and without activgate k8s-monitoring", func(t *testing.T) {
		dk := createStandaloneExtensionsDynakube(testDynakubeName, testApiUrl)
		dk.ObjectMeta.Annotations = map[string]string{
			exp.AGAutomaticK8sApiMonitoringKey: "false",
		}
		assertAllowedWithWarnings(t, 2, dk)
	})
}

func createStandaloneExtensionsDynakube(name, apiUrl string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     apiUrl,
			Extensions: &dynakube.ExtensionsSpec{},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
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
