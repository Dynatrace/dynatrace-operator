// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
)

// Component and pipeline IDs are typed constants so that MustNewID validates the names
// at program start, and the string representations derive from a single source.
var (
	prometheusID    = component.MustNewID("prometheus")
	memoryLimiterID = component.MustNewID("memory_limiter")
	otlpID          = component.MustNewID("otlp")
	healthCheckID   = component.MustNewID("health_check")
	metricsSignalID = pipeline.NewID(pipeline.SignalMetrics)
)

// scraperConfigData holds the inputs to the scraper's OTel Collector config that are
// derived from the owning DTPrometheus.
type scraperConfigData struct {
	// TargetAllocatorEndpoint is the URL the Prometheus receiver polls for its share
	// of the scrape targets.
	TargetAllocatorEndpoint string

	// GatewayEndpoint is the host:port the OTLP exporter forwards scraped metrics to.
	GatewayEndpoint string

	// TargetsPollInterval is how often the receiver re-polls the target allocator.
	TargetsPollInterval string
}

// buildScraperOTelConfig assembles the scraper's OTel Collector config as an
// otelcgen.Config. The caller marshals it to YAML via cfg.Marshal().
func buildScraperOTelConfig(data scraperConfigData) *otelcgen.Config {
	return &otelcgen.Config{
		Receivers: map[component.ID]component.Config{
			prometheusID: buildPrometheusReceiver(data),
		},
		Processors: map[component.ID]component.Config{
			memoryLimiterID: map[string]any{
				"check_interval":         "1s",
				"limit_percentage":       95,
				"spike_limit_percentage": 5,
			},
		},
		Exporters: map[component.ID]component.Config{
			otlpID: map[string]any{
				"endpoint": data.GatewayEndpoint,
				// The gateway is reached over the cluster-internal network; the hop is
				// not encrypted, so the exporter must not attempt a TLS handshake.
				"tls": map[string]any{"insecure": true},
			},
		},
		Extensions: map[component.ID]component.Config{
			healthCheckID: map[string]any{
				"endpoint": "${env:MY_POD_IP}:13133",
			},
		},
		Service: otelcgen.ServiceConfig{
			Extensions: extensions.Config{healthCheckID},
			Pipelines: pipelines.Config{
				metricsSignalID: &pipelines.PipelineConfig{
					Receivers:  []component.ID{prometheusID},
					Processors: []component.ID{memoryLimiterID},
					Exporters:  []component.ID{otlpID},
				},
			},
		},
	}
}

// buildPrometheusReceiver configures the Prometheus receiver in target-allocator mode:
// scrape targets are not listed statically but fetched from the target allocator, which
// assigns each scraper its share. collector_id identifies this pod in that assignment,
// so it must be unique and stable per pod.
func buildPrometheusReceiver(data scraperConfigData) component.Config {
	return map[string]any{
		// The receiver requires a config section even though every scrape config is
		// supplied by the target allocator.
		"config": map[string]any{
			"scrape_configs": []any{},
		},
		"target_allocator": map[string]any{
			"endpoint":     data.TargetAllocatorEndpoint,
			"interval":     data.TargetsPollInterval,
			"collector_id": "${env:MY_POD_NAME}",
		},
	}
}
