package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorOneAgentModuleDisabled      = installconfig.GetModuleValidationErrorMessage("OneAgent")
	errorActiveGateModuleDisabled    = installconfig.GetModuleValidationErrorMessage("ActiveGate")
	errorExtensionsModuleDisabled    = installconfig.GetModuleValidationErrorMessage("Extensions")
	errorLogMonitoringModuleDisabled = installconfig.GetModuleValidationErrorMessage("LogMonitoring")
)

func isOneAgentModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.NeedsOneAgent() && !v.modules.OneAgent {
		return errorOneAgentModuleDisabled
	}

	return ""
}

func isActiveGateModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() && !v.modules.ActiveGate {
		return errorActiveGateModuleDisabled
	}

	return ""
}

func isExtensionsModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.IsExtensionsEnabled() && !v.modules.Extensions {
		return errorExtensionsModuleDisabled
	}

	return ""
}

func isLogMonitoringModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsEnabled() && !v.modules.LogMonitoring {
		return errorLogMonitoringModuleDisabled
	}

	return ""
}
