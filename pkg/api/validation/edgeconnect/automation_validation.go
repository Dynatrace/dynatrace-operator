package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorAutomationRequiresProvisioner = `When enabling Kubernetes automation using provisioner mode is mandatory! Please enable spec.oauth.provisioner and provide a resp. OAuth client configuration.`
)

func automationRequiresProvisionerValidation(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	if ec.Spec.KubernetesAutomation != nil && ec.Spec.KubernetesAutomation.Enabled && !ec.Spec.OAuth.Provisioner {
		return errorAutomationRequiresProvisioner
	}

	return ""
}
