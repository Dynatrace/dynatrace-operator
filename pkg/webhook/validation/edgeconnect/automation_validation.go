package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
)

const (
	errorAutomationRequiresProvisioner = `When enabling Kubernetes automation using provisioner mode is mandatory! Please enable spec.oauth.provisioner and provide a resp. OAuth client configuration.
	`
)

func automationRequiresProvisionerValidation(_ context.Context, _ *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string {
	if edgeConnect.Spec.KubernetesAutomation != nil && edgeConnect.Spec.KubernetesAutomation.Enabled && !edgeConnect.Spec.OAuth.Provisioner {
		return errorAutomationRequiresProvisioner
	}

	return ""
}
