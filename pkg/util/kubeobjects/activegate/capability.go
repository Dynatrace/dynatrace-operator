package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
)

func SwitchCapability(dk *dynakube.DynaKube, capability activegate.Capability, wantEnabled bool) {
	hasEnabled := dk.ActiveGate().IsMode(capability.DisplayName)
	capabilities := &dk.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = removeCapability(*capabilities, capability.DisplayName)
	}
}

func removeCapability(capabilities []activegate.CapabilityDisplayName, removeMe activegate.CapabilityDisplayName) []activegate.CapabilityDisplayName {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}

	return capabilities
}
