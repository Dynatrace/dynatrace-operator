package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"golang.org/x/net/context"
)

const (
	defaultServiceAccountName               = "dynatrace-edgeconnect"
	errorKubernetesAutomationNotImplemented = `You tried to enable the EdgeConnect Kubernetes Automation feature and/or manually set a service account. This feature is still a work in progress and is not supported yet.`
)

func isKubernetesAutomationEnabled(_ context.Context, _ *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string {
	if edgeConnect.Spec.ServiceAccountName != defaultServiceAccountName || (edgeConnect.Spec.KubernetesAutomation != nil && edgeConnect.Spec.KubernetesAutomation.Enabled) {
		return errorKubernetesAutomationNotImplemented
	}

	return ""
}
