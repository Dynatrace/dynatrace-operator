package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
)

func TestKubernetesAutomation(t *testing.T) {
	t.Run("service account defined", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ServiceAccountName: testServiceAccountName,
			},
		}
		assertDeniedResponse(t, []string{errorKubernetesAutomationNotImplemented}, edgeConnect)
	})

	t.Run("kubernetes automation enabled", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ServiceAccountName: defaultServiceAccountName,
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{
					Enabled: true,
				},
			},
		}
		assertDeniedResponse(t, []string{errorKubernetesAutomationNotImplemented}, edgeConnect)
	})

	t.Run("kubernetes automation disabled", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer:          "id." + allowedSuffix[0],
				ServiceAccountName: defaultServiceAccountName,
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{
					Enabled: false,
				},
			},
		}
		assertAllowedResponse(t, edgeConnect)
	})
}
