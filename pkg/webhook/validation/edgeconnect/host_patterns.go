package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
)

const (
	errorHostPattersIsRequired = `hostPatters is required when using provisioner mode`
)

func checkHostPatternsValue(_ context.Context, _ *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string {
	if edgeConnect.Spec.OAuth.Provisioner && len(edgeConnect.Spec.HostPatterns) == 0 {
		return errorHostPattersIsRequired
	}
	return ""
}
