package telemetryservice

type Protocol string

const (
	nameSuffix = "-telemetry"

	OtlpProtocol   Protocol = "otlp"
	ZipkinProtocol Protocol = "zipkin"
	JaegerProtocol Protocol = "jaeger"
	StatsdProtocol Protocol = "statsd"
)

func KnownProtocols() []Protocol {
	return []Protocol{
		OtlpProtocol,
		ZipkinProtocol,
		JaegerProtocol,
		StatsdProtocol,
	}
}

func (spec *Spec) GetProtocols() []Protocol {
	if spec == nil {
		return []Protocol{}
	}

	if len(spec.Protocols) == 0 {
		return KnownProtocols()
	}

	protocols := make([]Protocol, len(spec.Protocols))
	for i, proto := range spec.Protocols {
		protocols[i] = Protocol(proto)
	}

	return protocols
}

func (ts *TelemetryService) SetName(name string) {
	ts.name = name
}

func (ts *TelemetryService) GetName() string {
	return ts.name + nameSuffix
}

func (ts *TelemetryService) IsEnabled() bool {
	return ts.Spec != nil
}
