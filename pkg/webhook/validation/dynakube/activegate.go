package dynakube

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	corev1 "k8s.io/api/core/v1"
)

const (
	errorConflictingActiveGateSections = `The DynaKube's specification tries to use the deprecated ActiveGate section(s) alongside the new ActiveGate section, which is not supported.
`

	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorDuplicateActiveGateCapability = `The DynaKube's specification tries to specify duplicate capabilities in the ActiveGate section, duplicate capability=%s.
Make sure you don't duplicate an Activegate capability in your custom resource.
`
	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`

	errorJoinedSyntheticActiveGateCapability = `The DynaKube's specification tries to specify both the synthetic capability along other (%v) capabilities.
The synthetic capability can't be configured alongside other capabilities in the same DynaKube. Try using a different DynaKubes, 1 for synthetic and 1 for the other capabilities.
`
)

func conflictingActiveGateConfiguration(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.DeprecatedActiveGateMode() && dynakube.ActiveGateMode() {
		log.Info("requested dynakube has conflicting active gate configuration", "name", dynakube.Name, "namespace", dynakube.Namespace)
		return errorConflictingActiveGateSections
	}
	return ""
}

func duplicateActiveGateCapabilities(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		capabilities := dynakube.Spec.ActiveGate.Capabilities
		duplicateChecker := map[dynatracev1beta1.CapabilityDisplayName]bool{}
		for _, capability := range capabilities {
			if duplicateChecker[capability] {
				log.Info("requested dynakube has duplicates in the active gate capabilities section", "name", dynakube.Name, "namespace", dynakube.Namespace)
				return fmt.Sprintf(errorDuplicateActiveGateCapability, capability)
			}
			duplicateChecker[capability] = true
		}
	}
	return ""
}

func invalidActiveGateCapabilities(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		capabilities := dynakube.Spec.ActiveGate.Capabilities
		for _, capability := range capabilities {
			if _, ok := dynatracev1beta1.ActiveGateDisplayNames[capability]; !ok {
				log.Info("requested dynakube has invalid active gate capability", "name", dynakube.Name, "namespace", dynakube.Namespace)
				return fmt.Sprintf(errorInvalidActiveGateCapability, capability)
			}
		}
	}
	return ""
}

func missingActiveGateMemoryLimit(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() &&
		!dynakube.IsSyntheticMonitoringEnabled() &&
		!memoryLimitSet(dynakube.Spec.ActiveGate.Resources) {
		return warningMissingActiveGateMemoryLimit
	}
	return ""
}

func memoryLimitSet(resources corev1.ResourceRequirements) bool {
	return resources.Limits != nil && resources.Limits.Memory() != nil
}

func exclusiveSyntheticCapability(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsSyntheticMonitoringEnabled() && len(dynakube.Spec.ActiveGate.Capabilities) > 0 {
		log.Info(
			"requested dynakube has the synthetic active gate capability accompanied with others",
			"name", dynakube.Name,
			"namespace", dynakube.Namespace)
		return fmt.Sprintf(errorJoinedSyntheticActiveGateCapability, dynakube.Spec.ActiveGate.Capabilities)
	}
	return ""
}
