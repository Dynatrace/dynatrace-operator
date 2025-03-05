package otelcgen

import (
	"slices"

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

	allowedPipelinesLogsReceiversIDs = []component.ID{OtlpID}

	// based on
	// stasd https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/statsdreceiver/factory.go#L33
	allowedPipelinesMetricsReceiversIDs = []component.ID{OtlpID, StatsdID}

	// based on
	// zipkin https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/zipkinreceiver/factory.go#L24
	// jaeger https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/jaegerreceiver/factory.go
	allowedPipelinesTracesReceiversIDs = []component.ID{OtlpID, JaegerID, ZipkinID}
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
				Receivers:  c.buildPipelinesReceivers(allowedPipelinesTracesReceiversIDs),
				Processors: append(buildProcessors(), batchTraces),
				Exporters:  buildExporters(),
			},
			metrics: &pipelines.PipelineConfig{
				Receivers:  c.buildPipelinesReceivers(allowedPipelinesMetricsReceiversIDs),
				Processors: append(buildProcessors(), cumulativeToDelta, batchMetrics),
				Exporters:  buildExporters(),
			},
			logs: &pipelines.PipelineConfig{
				Receivers:  c.buildPipelinesReceivers(allowedPipelinesLogsReceiversIDs),
				Processors: append(buildProcessors(), batchLogs),
				Exporters:  buildExporters(),
			},
		},
	}
}

func (c *Config) buildPipelinesReceivers(allowed []component.ID) []component.ID {
	return filter(c.protocolsToIDs(), func(id component.ID) bool {
		return slices.Contains(allowed, id)
	})
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

func filter(componentIDs []component.ID, f func(component.ID) bool) []component.ID {
	filtered := make([]component.ID, 0)

	for _, componentID := range componentIDs {
		if f(componentID) {
			filtered = append(filtered, componentID)
		}
	}

	return filtered
}
