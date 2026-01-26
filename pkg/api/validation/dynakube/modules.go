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

func isOneAgentModuleDisabled(_ context.Context, vc *validatorClient, dk *dynakube.DynaKube) string {
	if dk.OneAgent().IsDaemonsetRequired() && !vc.modules.OneAgent {
		return errorOneAgentModuleDisabled
	}

	return ""
}

func isActiveGateModuleDisabled(_ context.Context, vc *validatorClient, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() && !vc.modules.ActiveGate {
		return errorActiveGateModuleDisabled
	}

	if dk.ActiveGate().IsKubernetesMonitoringEnabled() && !vc.modules.KubernetesMonitoring {
		return errorKubernetesMonitoringModuleDisabled
	}

	return ""
}

func isExtensionsModuleDisabled(_ context.Context, vc *validatorClient, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsAnyEnabled() && !vc.modules.Extensions {
		return errorExtensionsModuleDisabled
	}

	return ""
}

func isLogMonitoringModuleDisabled(_ context.Context, vc *validatorClient, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsEnabled() && !vc.modules.LogMonitoring {
		return errorLogMonitoringModuleDisabled
	}

	return ""
}

func isKSPMDisabled(_ context.Context, vc *validatorClient, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() {
		errs := []string{}
		if !vc.modules.KSPM {
			errs = append(errs, errorKSPMModuleDisabled)
		}

		if !vc.modules.KubernetesMonitoring {
			errs = append(errs, errorKSPMDependsOnKubernetesMonitoringModule)
		}

		return strings.Join(errs, ",")
	}

	return ""
}
