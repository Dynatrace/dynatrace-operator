package otelcgen

import (
	"go.opentelemetry.io/collector/component"
)

var (
	otlphttp = component.MustNewID("otlphttp")
)

func (c *Config) buildExporters() map[component.ID]component.Config {
	return map[component.ID]component.Config{
		otlphttp: &ServerConfig{
			Endpoint: "test",
			// if in-cluster AG
			TLSSetting: &TLSSetting{
				CAFile: "/run/opensignals/cacerts/certs",
			},
			Headers: map[string]string{
				"Authorization": "Api-Token ${env:DT_API_TOKEN}",
			},
		},
	}
}
