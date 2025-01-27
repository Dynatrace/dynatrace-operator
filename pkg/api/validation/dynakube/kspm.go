package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorTooManyAGReplicas  = `The Dynakube's specification specifies KSPM, but has more than one ActiveGate replica. Only one ActiveGate replica is allowed in combination with KSPM.`
	errorKSPMMissingKubemon = `The Dynakube's specification specifies KSPM, but "kubernetes-monitoring" is not enabled on the Activegate.`
	errorKSPMMissingImage   = `The Dynakube's specification specifies KSPM, but no image repository/tag is configured.`
)

func tooManyAGReplicas(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && dk.ActiveGate().GetReplicas() > 1 {
		return errorTooManyAGReplicas
	}

	return ""
}

func missingKSPMDependency(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FeatureAutomaticKubernetesApiMonitoring()) {
		return errorKSPMMissingKubemon
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
