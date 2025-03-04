package telemetryservice

import "github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"

const (
	nameSuffix = "-telemetry"
)

func KnownProtocols() []otelcgen.Protocol {
	return []otelcgen.Protocol{
		otelcgen.OtlpProtocol,
		otelcgen.ZipkinProtocol,
		otelcgen.JaegerProtocol,
		otelcgen.StatsdProtocol,
	}
}

func (spec *Spec) GetProtocols() []otelcgen.Protocol {
	if spec == nil {
		return []otelcgen.Protocol{}
	}

	if len(spec.Protocols) == 0 {
		return KnownProtocols()
	}

	protocols := make([]otelcgen.Protocol, len(spec.Protocols))
	for i, proto := range spec.Protocols {
		protocols[i] = otelcgen.Protocol(proto)
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
