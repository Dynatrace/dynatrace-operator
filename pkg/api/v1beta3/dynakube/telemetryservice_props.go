package dynakube

type TelemetryServiceProtocol string

const (
	TelemetryServiceOtlpProtocol   TelemetryServiceProtocol = "otlp"
	TelemetryServiceZipkinProtocol TelemetryServiceProtocol = "zipkin"
	TelemetryServiceJaegerProtocol TelemetryServiceProtocol = "jaeger"
	TelemetryServiceStatsdProtocol TelemetryServiceProtocol = "statsd"
)

func TelemetryServiceKnownProtocols() []TelemetryServiceProtocol {
	return []TelemetryServiceProtocol{
		TelemetryServiceOtlpProtocol,
		TelemetryServiceZipkinProtocol,
		TelemetryServiceJaegerProtocol,
		TelemetryServiceStatsdProtocol,
	}
}

func (dk *DynaKube) TelemetryServiceProtocols() []TelemetryServiceProtocol {
	if dk.Spec.TelemetryService == nil {
		return []TelemetryServiceProtocol{}
	}

	if len(dk.Spec.TelemetryService.Protocols) == 0 {
		return TelemetryServiceKnownProtocols()
	}

	var protocols []TelemetryServiceProtocol
	for _, proto := range dk.Spec.TelemetryService.Protocols {
		protocols = append(protocols, TelemetryServiceProtocol(proto))
	}
	return protocols
}

func (dk *DynaKube) IsTelemetryServiceEnabled() bool {
	return dk.Spec.TelemetryService != nil
}
