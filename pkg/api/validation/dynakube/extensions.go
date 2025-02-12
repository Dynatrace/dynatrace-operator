package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningExtensionsWithoutK8SMonitoring = "The Dynakube is configured with extensions without an ActiveGate with `kubernetes-monitoring` enabled or the `automatic-kubernetes-monitoring` feature flag. You need to ensure that Kubernetes monitoring is setup for this cluster."
)

func extensionsWithoutK8SMonitoring(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.IsExtensionsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FeatureAutomaticKubernetesApiMonitoring()) {
		return warningExtensionsWithoutK8SMonitoring
	}

	return ""
}
