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
	loadBalancingID = component.MustNewID("load_balancing")
	healthCheckID   = component.MustNewID("health_check")
	metricsSignalID = pipeline.NewID(pipeline.SignalMetrics)
)

// scraperConfigData holds the inputs to the scraper's OTel Collector config that are
// derived from the owning DTPrometheus.
type scraperConfigData struct {
	// TargetAllocatorEndpoint is the URL the Prometheus receiver polls for its share
	// of the scrape targets.
	TargetAllocatorEndpoint string

	// GatewayService is the "<name>.<namespace>" service reference passed to the
	// load-balancing exporter's k8s resolver. It must not include the FQDN suffix
	// or port — those are expressed separately in the exporter config.
	GatewayService string

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
			loadBalancingID: map[string]any{
				"protocol": map[string]any{
					// The gateway is reached over the cluster-internal network; the hop is
					// not encrypted, so the exporter must not attempt a TLS handshake.
					"otlp": map[string]any{
						"tls": map[string]any{"insecure": true},
					},
				},
				"resolver": map[string]any{
					"k8s": map[string]any{
						"service": data.GatewayService,
						"ports":   []int{gatewayOTLPPort},
					},
				},
				"retry_on_failure": map[string]any{
					"enabled":          true,
					"initial_interval": "5s",
					"max_interval":     "30s",
					"max_elapsed_time": "300s",
				},
				"routing_key": "resource",
				"sending_queue": map[string]any{
					"batch": map[string]any{
						"flush_timeout": "10s",
						"max_size":      5000,
						"min_size":      500,
					},
					"enabled":       true,
					"num_consumers": 10,
					"queue_size":    10000,
				},
				"timeout": "10s",
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
					Exporters:  []component.ID{loadBalancingID},
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
		"target_allocator": map[string]any{
			"endpoint":     data.TargetAllocatorEndpoint,
			"interval":     data.TargetsPollInterval,
			"collector_id": "${env:MY_POD_NAME}",
		},
	}
}
