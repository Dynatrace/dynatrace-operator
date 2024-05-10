package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
)

const (
	errorHostPattersIsRequired = `hostPatterns is required when using provisioner mode`
)

func checkHostPatternsValue(_ context.Context, _ *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string {
	if !edgeConnect.IsK8SAutomationEnabled() && edgeConnect.IsProvisionerModeEnabled() && len(edgeConnect.Spec.HostPatterns) == 0 {
		return errorHostPattersIsRequired
	}

	return ""
}
