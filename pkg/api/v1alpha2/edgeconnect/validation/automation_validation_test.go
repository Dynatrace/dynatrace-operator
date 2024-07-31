package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAutomationValidator(t *testing.T) {
	t.Run("accept edgeconnect config without automation or oauth configs set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{},
		}
		require.Empty(t, automationRequiresProvisionerValidation(context.Background(), nil, ec))
	})
	t.Run("reject automation without provisioning", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{

			Spec: edgeconnect.EdgeConnectSpec{
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{Enabled: true},
				OAuth: edgeconnect.OAuthSpec{
					Provisioner: false,
				},
				Resources: corev1.ResourceRequirements{},
			},
		}
		require.Equal(t, errorAutomationRequiresProvisioner, automationRequiresProvisionerValidation(context.Background(), nil, ec))
	})
	t.Run("accept automation with provisioning enabled", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{

			Spec: edgeconnect.EdgeConnectSpec{
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{Enabled: true},
				OAuth: edgeconnect.OAuthSpec{
					Provisioner: true,
				},
			},
		}
		require.Empty(t, automationRequiresProvisionerValidation(context.Background(), nil, ec))
	})
	t.Run("reject automation with no oauth config", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{

			Spec: edgeconnect.EdgeConnectSpec{
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{Enabled: true},
			},
		}
		require.Equal(t, errorAutomationRequiresProvisioner, automationRequiresProvisionerValidation(context.Background(), nil, ec))
	})
}
