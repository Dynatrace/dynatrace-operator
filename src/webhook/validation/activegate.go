package validation

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorDuplicateActiveGateCapability = `The DynaKube's specification tries to specify duplicate capabilities in the ActiveGate section, duplicate capability=%s.
Make sure you don't duplicate an Activegate capability in your custom resource.
`
	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`

	errorJoinedSyntheticActiveGateCapability = `The DynaKube's specification requires illegally the synthetic capability along with %v.
Make sure such a capability is the single one.
`
)

func conflictingActiveGateConfiguration(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return ""
}

func duplicateActiveGateCapabilities(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
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

func invalidActiveGateCapabilities(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
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

func missingActiveGateMemoryLimit(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() &&
		!dynakube.IsSyntheticActiveGateEnabled() &&
		!memoryLimitSet(dynakube.Spec.ActiveGate.Resources) {
		return warningMissingActiveGateMemoryLimit
	}
	return ""
}

func memoryLimitSet(resources corev1.ResourceRequirements) bool {
	return resources.Limits != nil && resources.Limits.Memory() != nil
}

func exclusiveSyntheticCapability(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsSyntheticActiveGateEnabled() && len(dynakube.Spec.ActiveGate.Capabilities) > 1 {
		log.Info(
			"requested dynakube has the synthetic active gate capability accompanied with others",
			"name", dynakube.Name,
			"namespace", dynakube.Namespace)
		return fmt.Sprintf(errorJoinedSyntheticActiveGateCapability, syntheticlessCapabilities(dynakube))
	}
	return ""
}

func syntheticlessCapabilities(dynakube *dynatracev1beta1.DynaKube) string {
	const separator = ' '

	collected := strings.Builder{}
	collected.WriteRune('[')
	for _, capability := range dynakube.Spec.ActiveGate.Capabilities {
		if capability != dynatracev1beta1.SyntheticCapability.DisplayName {
			if collected.Len() > 1 {
				collected.WriteRune(separator)
			}
			collected.WriteString(string(capability))
		}
	}
	collected.WriteRune(']')

	return collected.String()
}
