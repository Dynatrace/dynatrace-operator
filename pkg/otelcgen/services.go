package otelcgen

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service"
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

func (c *Config) buildServices() service.Config {
	return service.Config{
		Telemetry: telemetry.Config{
			Logs:     telemetry.LogsConfig{},
			Metrics:  telemetry.MetricsConfig{},
			Traces:   telemetry.TracesConfig{},
			Resource: nil,
		},
		Extensions: extensions.Config{healthCheck},
		Pipelines: pipelines.Config{
			traces: &pipelines.PipelineConfig{
				Receivers:  buildReceivers(),
				Processors: append(buildProcessors(), batchTraces),
				Exporters:  buildExporters(),
			},
			metrics: &pipelines.PipelineConfig{
				Receivers:  buildReceivers(),
				Processors: append(buildProcessors(), batchMetrics),
				Exporters:  buildExporters(),
			},
			logs: &pipelines.PipelineConfig{
				Receivers:  buildReceivers(),
				Processors: append(buildProcessors(), batchLogs),
				Exporters:  buildExporters(),
			},
		},
	}
}

func buildReceivers() []component.ID {
	return []component.ID{
		OtlpID, JaegerID, ZipkinID,
	}
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
