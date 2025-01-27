package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorExtensionsWithoutK8SMonitoring = "The Dynakube's specification enables extensions without an ActiveGate which has Kubernetes monitoring enabled. This is not feasible, as the cluster will not be visible in Dynatrace without the Kubernetes monitoring feature."
)

func extensionsWithoutK8SMonitoring(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.IsExtensionsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FeatureAutomaticKubernetesApiMonitoring()) {
		return errorExtensionsWithoutK8SMonitoring
	}

	return ""
}
