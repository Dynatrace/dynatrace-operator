package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorCSIModuleRequired           = "CSI Driver was disabled during Operator install. It is a necessary resource for Cloud Native Fullstack to work. Redeploy the Operator via Helm with the CSI Driver enabled."
	errorOneAgentModuleDisabled      = installconfig.GetModuleValidationErrorMessage("OneAgent")
	errorActiveGateModuleDisabled    = installconfig.GetModuleValidationErrorMessage("ActiveGate")
	errorExtensionsModuleDisabled    = installconfig.GetModuleValidationErrorMessage("Extensions")
	errorLogMonitoringModuleDisabled = installconfig.GetModuleValidationErrorMessage("LogMonitoring")
	errorKSPMDisabled                = installconfig.GetModuleValidationErrorMessage("KSPM")
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

func isKSPMDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && !v.modules.KSPM {
		return errorKSPMDisabled
	}

	return ""
}

func isCSIModuleDisabled(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if isCSIRequired(dk) && !v.modules.CSIDriver {
		return errorCSIModuleRequired
	}

	return ""
}

// isCSIRequired checks if the provided a DynaKube strictly needs the csi-driver, and no fallbacks exist to provide the same functionality.
func isCSIRequired(dk *dynakube.DynaKube) bool {
	return dk.CloudNativeFullstackMode()
}
