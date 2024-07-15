package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func SwitchCapability(dk *dynakube.DynaKube, capability dynakube.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := dk.IsActiveGateMode(capability.DisplayName)
	capabilities := &dk.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = removeCapability(*capabilities, capability.DisplayName)
	}
}

func removeCapability(capabilities []dynakube.CapabilityDisplayName, removeMe dynakube.CapabilityDisplayName) []dynakube.CapabilityDisplayName {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}

	return capabilities
}
