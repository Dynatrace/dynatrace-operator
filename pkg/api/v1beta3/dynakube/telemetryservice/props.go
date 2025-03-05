package telemetryservice

import "github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"

const (
	nameSuffix = "-telemetry"
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

func (ts *TelemetryService) SetName(name string) {
	ts.name = name
}

func (ts *TelemetryService) GetName() string {
	return ts.name + nameSuffix
}

func (ts *TelemetryService) IsEnabled() bool {
	return ts.Spec != nil
}
