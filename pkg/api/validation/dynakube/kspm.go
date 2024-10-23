package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorKSPMMissingKubemon = `For the KSPM feature, the "kubernetes-monitoring" capability also needs to be enabled on the ActiveGate. `
)

func missingKSPMDependency(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() &&
		!dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		return errorKSPMMissingKubemon
	}

	return ""
}
