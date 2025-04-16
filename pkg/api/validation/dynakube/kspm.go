package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
)

const (
	errorTooManyAGReplicas    = `The Dynakube's specification specifies KSPM, but has more than one ActiveGate replica. Only one ActiveGate replica is allowed in combination with KSPM.`
	warningKSPMMissingKubemon = "The Dynakube is configured with KSPM without an ActiveGate with `kubernetes-monitoring` enabled or the `automatic-kubernetes-monitoring` feature flag. You need to ensure that Kubernetes monitoring is setup for this cluster."
	errorKSPMMissingImage     = `The Dynakube's specification specifies KSPM, but no image repository/tag is configured.`
)

func tooManyAGReplicas(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && dk.ActiveGate().GetReplicas() > 1 {
		return errorTooManyAGReplicas
	}

	return ""
}

func kspmWithoutK8SMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FF().IsAutomaticK8sApiMonitoring()) {
		return warningKSPMMissingKubemon
	}

	return ""
}

func missingKSPMImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	if dk.KSPM().ImageRef.Repository == "" || dk.KSPM().ImageRef.Tag == "" {
		return errorKSPMMissingImage
	}

	return ""
}
