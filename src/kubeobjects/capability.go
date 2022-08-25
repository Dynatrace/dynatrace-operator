package kubeobjects

import dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"

func SwitchCapability(instance *dynatracev1beta1.DynaKube, capability dynatracev1beta1.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := instance.IsActiveGateMode(capability.DisplayName)
	capabilities := &instance.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = removeCapability(*capabilities, capability.DisplayName)
	}
}

func removeCapability(capabilities []dynatracev1beta1.CapabilityDisplayName, removeMe dynatracev1beta1.CapabilityDisplayName) []dynatracev1beta1.CapabilityDisplayName {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}
	return capabilities
}
