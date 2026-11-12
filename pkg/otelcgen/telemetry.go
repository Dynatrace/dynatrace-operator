// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
)

// NewMetricsTelemetry returns the collector's own telemetry config (service::telemetry) exporting
// metrics via OTLP/HTTP with delta temporality to the given base URL, e.g. "http://host:4318".
// The OTLP metrics path is appended by the SDK.
//
// It starts from the collector's default telemetry config so that settings that are marshaled
// explicitly (e.g. logs) keep their defaults instead of being overridden by zero values.
func NewMetricsTelemetry(endpoint string) otelconftelemetry.Config {
	cfg := *otelconftelemetry.NewFactory().CreateDefaultConfig().(*otelconftelemetry.Config)

	cfg.Metrics.Readers = []otelconf.MetricReader{{
		Periodic: &otelconf.PeriodicMetricReader{
			Exporter: otelconf.PushMetricExporter{
				OTLP: &otelconf.OTLPMetric{
					Protocol:              new("http/protobuf"),
					Endpoint:              &endpoint,
					TemporalityPreference: new("delta"),
				},
			},
		},
	}}

	return cfg
}
