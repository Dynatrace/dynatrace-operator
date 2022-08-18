package testinghelpers

import dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"

type DtCapability = dynatracev1beta1.CapabilityDisplayName

func DoTestSetCapability(instance *dynatracev1beta1.DynaKube, capability dynatracev1beta1.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := instance.IsActiveGateMode(capability.DisplayName)
	capabilities := &instance.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = testRemoveCapability(*capabilities, capability.DisplayName)
	}
}

func testRemoveCapability(capabilities []DtCapability, removeMe DtCapability) []DtCapability {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}
	return capabilities
}
