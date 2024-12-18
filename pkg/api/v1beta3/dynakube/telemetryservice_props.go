package dynakube

const (
	TelemetryServiceOtlpProtocol   = "otlp"
	TelemetryServiceZipkinProtocol = "zipkin"
	TelemetryServiceJaegerProtocol = "jaeger"
	TelemetryServiceStatsdProtocol = "statsd"
)

func TelemetryServiceKnownProtocols() []string {
	return []string{
		TelemetryServiceOtlpProtocol,
		TelemetryServiceZipkinProtocol,
		TelemetryServiceJaegerProtocol,
		TelemetryServiceStatsdProtocol,
	}
}

func (dk *DynaKube) TelemetryServiceProtocols() []string {
	if dk.Spec.TelemetryService == nil {
		return []string{}
	}

	if dk.Spec.TelemetryService.Protocols == nil {
		return TelemetryServiceKnownProtocols()
	}

	return dk.Spec.TelemetryService.Protocols
}

func (dk *DynaKube) IsTelemetryServiceEnabled() bool {
	return dk.Spec.TelemetryService != nil
}
