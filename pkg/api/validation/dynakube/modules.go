package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorOneAgentModuleDisabled   = `OneAgent features has been disabled during Operator install. The necessary resources for deploying a OneAgent DaemonSet to work are not present on the cluster. Redeploy the Operator with all the necessary resources`
	errorActiveGateModuleDisabled = `ActiveGate features has been disabled during Operator install. The necessary resources for deploying a ActiveGate Statefulset to work are not present on the cluster. Redeploy the Operator with all the necessary resources`
)

func isOneAgentModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.NeedsOneAgent() && !v.modules.OneAgent {
		return errorOneAgentModuleDisabled
	}

	return ""
}

func isActiveGateModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.NeedsActiveGate() && !v.modules.ActiveGate {
		return errorActiveGateModuleDisabled
	}

	return ""
}
