package validation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	corev1 "k8s.io/api/core/v1"
)

const (
	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorActiveGateInvalidPVCConfiguration = ` DynaKube specifies a PVC for the ActiveGate while ephemeral volume is also enabled. These settings are mutually exclusive, please choose only one.`

	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`

	warningActiveGateRollingUpdateOldK8sVersion = `ActiveGate rollingUpdate setting requires Kubernetes version 1.35 or higher. The current cluster version is below 1.35, so the rollingUpdate setting will be ignored.`

	minK8sMinorVersionForRollingUpdate = 35
)

func invalidActiveGateCapabilities(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() {
		capabilities := dk.Spec.ActiveGate.Capabilities
		for _, capability := range capabilities {
			if _, ok := activegate.CapabilityDisplayNames[capability]; !ok {
				log.Info("requested dynakube has invalid active gate capability", "name", dk.Name, "namespace", dk.Namespace)

				return fmt.Sprintf(errorInvalidActiveGateCapability, capability)
			}
		}
	}

	return ""
}

func missingActiveGateMemoryLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() &&
		!memoryLimitSet(dk.Spec.ActiveGate.Resources) {
		return warningMissingActiveGateMemoryLimit
	}

	return ""
}

func memoryLimitSet(resources corev1.ResourceRequirements) bool {
	return resources.Limits != nil && resources.Limits.Memory() != nil
}

func activeGateMutuallyExclusivePVCSettings(dk *dynakube.DynaKube) bool {
	return dk.Spec.ActiveGate.UseEphemeralVolume && dk.Spec.ActiveGate.VolumeClaimTemplate != nil
}

func mutuallyExclusiveActiveGatePVsettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if activeGateMutuallyExclusivePVCSettings(dk) {
		log.Info("requested dynakube specifies mutually exclusive VolumeClaimTemplate settings for ActiveGate.", "name", dk.Name, "namespace", dk.Namespace)

		return errorActiveGateInvalidPVCConfiguration
	}

	return ""
}

func activeGateRollingUpdateWithOldK8sVersion(_ context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.ActiveGate.RollingUpdate == nil {
		return ""
	}

	if dv.versionChecker == nil {
		return ""
	}

	serverVersion, err := dv.versionChecker.ServerVersion()
	if err != nil {
		log.Error(err, "failed to get kubernetes server version")
		return ""
	}

	minor, err := strconv.Atoi(serverVersion.Minor)
	if err != nil {
		log.Error(err, "failed to parse kubernetes minor version", "minor", serverVersion.Minor)
		return ""
	}

	if minor < minK8sMinorVersionForRollingUpdate {
		return warningActiveGateRollingUpdateOldK8sVersion
	}

	return ""
}
