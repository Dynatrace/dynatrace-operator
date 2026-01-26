package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	corev1 "k8s.io/api/core/v1"
)

const (
	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorDuplicateActiveGateCapability = `The DynaKube's specification tries to specify duplicate capabilities in the ActiveGate section, duplicate capability=%s.
Make sure you don't duplicate an Activegate capability in your custom resource.
`
	errorActiveGateInvalidPVCConfiguration = ` DynaKube specifies a PVC for the ActiveGate while ephemeral volume is also enabled. These settings are mutually exclusive, please choose only one.`

	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`
)

func duplicateActiveGateCapabilities(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() {
		capabilities := dk.Spec.ActiveGate.Capabilities
		duplicateChecker := map[activegate.CapabilityDisplayName]bool{}

		for _, capability := range capabilities {
			if duplicateChecker[capability] {
				log.Info("requested dynakube has duplicates in the active gate capabilities section", "name", dk.Name, "namespace", dk.Namespace)

				return fmt.Sprintf(errorDuplicateActiveGateCapability, capability)
			}

			duplicateChecker[capability] = true
		}
	}

	return ""
}

func invalidActiveGateCapabilities(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
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

func missingActiveGateMemoryLimit(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
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

func mutuallyExclusiveActiveGatePVsettings(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if activeGateMutuallyExclusivePVCSettings(dk) {
		log.Info("requested dynakube specifies mutually exclusive VolumeClaimTemplate settings for ActiveGate.", "name", dk.Name, "namespace", dk.Namespace)

		return errorActiveGateInvalidPVCConfiguration
	}

	return ""
}
