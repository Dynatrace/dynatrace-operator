// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package telemetryingest

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcfg"
)

const (
	ServiceNameSuffix = "-telemetry-ingest"
)

func (spec *Spec) GetProtocols() []otelcfg.Protocol {
	if spec == nil {
		return []otelcfg.Protocol{}
	}

	if len(spec.Protocols) == 0 {
		return otelcfg.RegisteredProtocols
	}

	return spec.Protocols
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

func (ts *TelemetryIngest) IsOTLPEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	return slices.Contains(ts.GetProtocols(), otelcfg.OTLPProtocol)
}

func (ts *TelemetryIngest) IsJaegerEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	return slices.Contains(ts.GetProtocols(), otelcfg.JaegerProtocol)
}

func (ts *TelemetryIngest) IsZipkinEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	return slices.Contains(ts.GetProtocols(), otelcfg.ZipkinProtocol)
}

func (ts *TelemetryIngest) IsStatsdEnabled() bool {
	if !ts.IsEnabled() {
		return false
	}

	return slices.Contains(ts.GetProtocols(), otelcfg.StatsdProtocol)
}
