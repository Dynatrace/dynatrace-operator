package kubeobjects

import dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"

func SwitchCapability(instance *dynatracev1.DynaKube, capability dynatracev1.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := instance.IsActiveGateMode(capability.DisplayName)
	capabilities := &instance.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = removeCapability(*capabilities, capability.DisplayName)
	}
}

func removeCapability(capabilities []dynatracev1.CapabilityDisplayName, removeMe dynatracev1.CapabilityDisplayName) []dynatracev1.CapabilityDisplayName {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}
	return capabilities
}
