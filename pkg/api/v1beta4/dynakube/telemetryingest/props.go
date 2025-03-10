package telemetryingest

import "github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"

const (
	ServiceNameSuffix = "-telemetry-ingest"
)

func (spec *Spec) GetProtocols() otelcgen.Protocols {
	if spec == nil {
		return otelcgen.Protocols{}
	}

	if len(spec.Protocols) == 0 {
		return otelcgen.RegisteredProtocols
	}

	protocols := make(otelcgen.Protocols, len(spec.Protocols))
	for i, proto := range spec.Protocols {
		protocols[i] = otelcgen.Protocol(proto)
	}

	return protocols
}

func (ts *TelemetryIngest) SetName(name string) {
	ts.name = name
}

func (ts *TelemetryIngest) GetName() string {
	return ts.name + ServiceNameSuffix
}

func (ts *TelemetryIngest) IsEnabled() bool {
	return ts.Spec != nil
}
