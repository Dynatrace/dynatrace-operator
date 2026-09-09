// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

var (
	traces  = pipeline.NewID(pipeline.SignalTraces)
	metrics = pipeline.NewID(pipeline.SignalMetrics)
	logs    = pipeline.NewID(pipeline.SignalLogs)

	allowedPipelinesLogsReceiversIDs = []component.ID{OTLPID}

	// based on
	// stasd https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/statsdreceiver/factory.go#L33
	allowedPipelinesMetricsReceiversIDs = []component.ID{OTLPID, StatsdID}

	// based on
	// zipkin https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/zipkinreceiver/factory.go#L24
	// jaeger https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/d4372922ec79cb052c7f7e2fcc0fba9f492bd948/receiver/jaegerreceiver/factory.go
	allowedPipelinesTracesReceiversIDs = []component.ID{OTLPID, JaegerID, ZipkinID}
)

// ServiceConfig defines the configurable components of the Service.
// based on "go.opentelemetry.io/collector/service.Config
type ServiceConfig struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry otelconftelemetry.Config `mapstructure:"telemetry,omitempty"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions"`
}

func (c *Config) buildServices() ServiceConfig {
	pipelinesCfg := pipelines.Config{}

	// traces
	tracesReceivers := c.buildPipelinesReceivers(allowedPipelinesTracesReceiversIDs)
	if len(tracesReceivers) != 0 {
		pipelinesCfg[traces] = &pipelines.PipelineConfig{
			Receivers:  tracesReceivers,
			Processors: append(c.buildPipelineProcessors(), batchTraces),
			Exporters:  buildExporters(),
		}
	}

	// metrics
	metricsReceivers := c.buildPipelinesReceivers(allowedPipelinesMetricsReceiversIDs)
	if len(metricsReceivers) != 0 {
		pipelinesCfg[metrics] = &pipelines.PipelineConfig{
			Receivers:  metricsReceivers,
			Processors: append(c.buildPipelineProcessors(), cumulativeToDelta, batchMetrics),
			Exporters:  buildExporters(),
		}
	}

	// logs
	logsReceivers := c.buildPipelinesReceivers(allowedPipelinesLogsReceiversIDs)
	if len(logsReceivers) != 0 {
		pipelinesCfg[logs] = &pipelines.PipelineConfig{
			Receivers:  logsReceivers,
			Processors: append(c.buildPipelineProcessors(), batchLogs),
			Exporters:  buildExporters(),
		}
	}

	return ServiceConfig{
		Extensions: extensions.Config{healthCheck},
		Pipelines:  pipelinesCfg,
	}
}

func (c *Config) buildPipelinesReceivers(allowed []component.ID) []component.ID {
	return filter(c.protocolsToIDs(), func(id component.ID) bool {
		return slices.Contains(allowed, id)
	})
}

func buildExporters() []component.ID {
	return []component.ID{
		otlphttp,
	}
}

func (c *Config) buildPipelineProcessors() []component.ID {
	// k8sattributesAnnotations must run before staticResourceAttrs so per-pod annotations claim
	// their keys first (k8sattributesprocessor never overwrites an already-set attribute, so this
	// is the only way for it to win); staticResourceAttrs must in turn run before
	// k8sattributesFacts so the static DynaKube default can claim a key before the built-in fact
	// extraction would. transform runs last: its merge_maps statement unpacks the
	// metadata.dynatrace.com JSON blob (written by the enrichment webhook, which already resolves
	// the full precedence chain) with "upsert", so that blob always wins over both static and facts.
	processors := []component.ID{memoryLimiter, transformPodIP, k8sattributesAnnotations}

	if len(c.resourceAttributes) > 0 {
		processors = append(processors, staticResourceAttrs)
	}

	return append(processors, k8sattributesFacts, transform)
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
