package otelcgen

import (
	"go.opentelemetry.io/collector/component"
)

var (
	otlphttp = component.MustNewID("otlphttp")
)

func (c *Config) buildExporters() map[component.ID]component.Config {
	serverConfig := &ServerConfig{
		Endpoint:   c.buildEndpointWithoutPort(),
		TLSSetting: &TLSSetting{},
	}

	if c.caFile != "" {
		serverConfig.TLSSetting.CAFile = c.caFile
	}

	if c.apiToken != "" {
		serverConfig.Headers = make(map[string]string)
		serverConfig.Headers["Authorization"] = "Api-Token " + c.apiToken
	}

	return map[component.ID]component.Config{
		otlphttp: serverConfig,
	}
}
