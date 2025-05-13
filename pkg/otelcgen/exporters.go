package otelcgen

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtls"
)

var (
	otlphttp = component.MustNewID("otlphttp")
)

func (c *Config) buildExporters() map[component.ID]component.Config {
	serverConfig := &ServerConfig{
		Endpoint: c.buildExportersEndpoint(),
	}

	if c.caFile != "" {
		serverConfig.TLSSetting = &TLSSetting{
			Config: configtls.Config{
				CAFile: c.caFile,
			},
		}
		if c.includeSystemCACertsPool {
			serverConfig.TLSSetting.IncludeSystemCACertsPool = c.includeSystemCACertsPool
		}
	}

	if c.apiToken != "" {
		serverConfig.Headers = make(map[string]string)
		serverConfig.Headers["Authorization"] = "Api-Token " + c.apiToken
	}

	return map[component.ID]component.Config{
		otlphttp: serverConfig,
	}
}
