// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

// SelfMonitoring sets the collector's own telemetry config (service::telemetry) exporting
// metrics via OTLP/HTTP with delta temporality to the given base URL, e.g. "http://host:4318".
// The OTLP metrics path is appended by the SDK.
func (b *Builder) SelfMonitoring(endpoint string) *Builder {
	if endpoint == "" {
		return b
	}

	// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/service/telemetry/otelconftelemetry/config.go#L14
	b.cfg.Service.Telemetry = map[string]any{
		"metrics": map[string]any{
			"level": "Normal",
			"readers": []any{
				map[string]any{
					"periodic": map[string]any{
						"exporter": map[string]any{
							"otlp": map[string]any{
								"endpoint":               endpoint,
								"protocol":               "http/protobuf",
								"temporality_preference": "delta",
							},
						},
					},
				},
			},
		},
	}

	return b
}
