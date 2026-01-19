package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningExtensionsWithoutK8SMonitoring = "The Dynakube is configured with extensions without an ActiveGate with `kubernetes-monitoring` enabled or the `automatic-kubernetes-api-monitoring` feature flag. You need to ensure that Kubernetes monitoring is setup for this cluster."
)

func extensionsWithoutK8SMonitoring(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FF().IsAutomaticK8sAPIMonitoring()) {
		return warningExtensionsWithoutK8SMonitoring
	}

	return ""
}
