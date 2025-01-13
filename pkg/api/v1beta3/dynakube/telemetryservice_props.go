package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"

func (dk *DynaKube) TelemetryServiceProtocols() []telemetryservice.Protocol {
	if dk.Spec.TelemetryService == nil {
		return []telemetryservice.Protocol{}
	}

	if len(dk.Spec.TelemetryService.Protocols) == 0 {
		return telemetryservice.KnownProtocols()
	}

	var protocols []telemetryservice.Protocol
	for _, proto := range dk.Spec.TelemetryService.Protocols {
		protocols = append(protocols, telemetryservice.Protocol(proto))
	}
	return protocols
}

func (dk *DynaKube) IsTelemetryServiceEnabled() bool {
	return dk.Spec.TelemetryService != nil
}
