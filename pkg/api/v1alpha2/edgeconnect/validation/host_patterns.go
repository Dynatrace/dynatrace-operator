package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorHostPattersIsRequired = `hostPatterns is required when using provisioner mode`
)

func checkHostPatternsValue(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	if ec.IsProvisionerModeEnabled() && len(ec.Spec.HostPatterns) == 0 && !ec.IsK8SAutomationEnabled() {
		return errorHostPattersIsRequired
	}

	return ""
}
