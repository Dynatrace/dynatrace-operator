package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorOneAgentModuleDisabled             = installconfig.GetModuleValidationErrorMessage("OneAgent")
	errorActiveGateModuleDisabled           = installconfig.GetModuleValidationErrorMessage("ActiveGate")
	errorExtensionsModuleDisabled           = installconfig.GetModuleValidationErrorMessage("Extensions")
	errorLogMonitoringModuleDisabled        = installconfig.GetModuleValidationErrorMessage("LogMonitoring")
	errorKSPMDisabled                       = installconfig.GetModuleValidationErrorMessage("KSPM")
	errorKubernetesMonitoringModuleDisabled = installconfig.GetModuleValidationErrorMessage("KubernetesMonitoring")
)

func isOneAgentModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.OneAgent().IsDaemonsetRequired() && !v.modules.OneAgent {
		return errorOneAgentModuleDisabled
	}

	return ""
}

func isActiveGateModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() && !v.modules.ActiveGate {
		return errorActiveGateModuleDisabled
	}

	if dk.ActiveGate().IsKubernetesMonitoringEnabled() && !v.modules.KubernetesMonitoring && !v.modules.KSPM {
		return errorKubernetesMonitoringModuleDisabled
	}

	return ""
}

func isExtensionsModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsEnabled() && !v.modules.Extensions {
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

func isKSPMDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && !v.modules.KSPM {
		return errorKSPMDisabled
	}

	return ""
}
