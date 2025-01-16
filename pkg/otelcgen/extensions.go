package otelcgen

import "go.opentelemetry.io/collector/component"

var (
	healthCheck = component.MustNewID("health_check")
)

func (c *Config) buildExtensions() map[component.ID]component.Config {
	return map[component.ID]component.Config{
		healthCheck: &ServerConfig{
			Endpoint: "test",
		},
	}
}
