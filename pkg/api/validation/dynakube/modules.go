package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorOneAgentModuleDisabled                  = installconfig.GetModuleValidationErrorMessage("OneAgent")
	errorActiveGateModuleDisabled                = installconfig.GetModuleValidationErrorMessage("ActiveGate")
	errorExtensionsModuleDisabled                = installconfig.GetModuleValidationErrorMessage("Extensions")
	errorLogMonitoringModuleDisabled             = installconfig.GetModuleValidationErrorMessage("LogMonitoring")
	errorKSPMModuleDisabled                      = installconfig.GetModuleValidationErrorMessage("KSPM")
	errorKubernetesMonitoringModuleDisabled      = installconfig.GetModuleValidationErrorMessage("KubernetesMonitoring")
	errorKSPMDependsOnKubernetesMonitoringModule = installconfig.GetDependentModuleValidationErrorMessage("KubernetesMonitoring", "KSPM")
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

	if dk.ActiveGate().IsKubernetesMonitoringEnabled() && !v.modules.KubernetesMonitoring {
		return errorKubernetesMonitoringModuleDisabled
	}

	return ""
}

func isExtensionsModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsAnyEnabled() && !v.modules.Extensions {
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
	if dk.KSPM().IsEnabled() {
		errs := []string{}
		if !v.modules.KSPM {
			errs = append(errs, errorKSPMModuleDisabled)
		}

		if !v.modules.KubernetesMonitoring {
			errs = append(errs, errorKSPMDependsOnKubernetesMonitoringModule)
		}

		return strings.Join(errs, ",")
	}

	return ""
}
