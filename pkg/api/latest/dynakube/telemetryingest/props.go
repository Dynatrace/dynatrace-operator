package telemetryingest

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
)

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

func (ts *TelemetryIngest) GetDefaultServiceName() string {
	return ts.name + ServiceNameSuffix
}

func (ts *TelemetryIngest) GetServiceName() string {
	if ts.Spec == nil {
		return ts.GetDefaultServiceName()
	}

	serviceName := ts.ServiceName
	if serviceName == "" {
		serviceName = ts.GetDefaultServiceName()
	}

	return serviceName
}

func (ts *TelemetryIngest) IsEnabled() bool {
	return ts.Spec != nil
}

func (ts *TelemetryIngest) IsOtlpEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	protocols := ts.GetProtocols()

	return slices.Contains(protocols, otelcgen.OtlpProtocol)
}

func (ts *TelemetryIngest) IsJaegerEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	protocols := ts.GetProtocols()

	return slices.Contains(protocols, otelcgen.JaegerProtocol)
}

func (ts *TelemetryIngest) IsZipkinEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	protocols := ts.GetProtocols()

	return slices.Contains(protocols, otelcgen.ZipkinProtocol)
}

func (ts *TelemetryIngest) IsStatsdEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	protocols := ts.GetProtocols()

	return slices.Contains(protocols, otelcgen.StatsdProtocol)
}
