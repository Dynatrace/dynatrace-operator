package otelcgen

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

var (
	traces  = pipeline.MustNewID("traces")
	metrics = pipeline.MustNewID("metrics")
	logs    = pipeline.MustNewID("logs")
	debug   = component.MustNewID("debug")
)

// ServiceConfig defines the configurable components of the Service.
// based on "go.opentelemetry.io/collector/service.Config
type ServiceConfig struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry,omitempty"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions"`
}

func (c *Config) buildServices() ServiceConfig {
	return ServiceConfig{
		Extensions: extensions.Config{healthCheck},
		Pipelines: pipelines.Config{
			traces: &pipelines.PipelineConfig{
				Receivers:  c.buildPipelinesReceivers(),
				Processors: append(buildProcessors(), batchTraces),
				Exporters:  buildExporters(),
			},
			metrics: &pipelines.PipelineConfig{
				Receivers:  c.buildPipelinesReceivers(),
				Processors: append(buildProcessors(), cumulativeToDelta, batchMetrics),
				Exporters:  buildExporters(),
			},
			logs: &pipelines.PipelineConfig{
				Receivers:  c.buildPipelinesReceivers(),
				Processors: append(buildProcessors(), batchLogs),
				Exporters:  buildExporters(),
			},
		},
	}
}

func (c *Config) buildPipelinesReceivers() []component.ID {
	return c.protocolsToIDs()
}

func buildExporters() []component.ID {
	return []component.ID{
		otlphttp, debug,
	}
}

func buildProcessors() []component.ID {
	return []component.ID{
		memoryLimiter, k8sattributes, transform,
	}
}
