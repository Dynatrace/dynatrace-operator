package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorKSPMMissingKubemon = `The Dynakube's specification specifies KSPM, but "kubernetes-monitoring" is not enabled on the Activegate.`
)

func missingKSPMDependency(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() &&
		!dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		return errorKSPMMissingKubemon
	}

	return ""
}
