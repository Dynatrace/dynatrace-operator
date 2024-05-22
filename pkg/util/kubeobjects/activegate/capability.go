package activegate

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
)

func SwitchCapability(instance *dynatracev1beta2.DynaKube, capability dynatracev1beta2.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := instance.IsActiveGateMode(capability.DisplayName)
	capabilities := &instance.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = removeCapability(*capabilities, capability.DisplayName)
	}
}

func removeCapability(capabilities []dynatracev1beta2.CapabilityDisplayName, removeMe dynatracev1beta2.CapabilityDisplayName) []dynatracev1beta2.CapabilityDisplayName {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}

	return capabilities
}
