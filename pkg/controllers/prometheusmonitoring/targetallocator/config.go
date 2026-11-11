// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config is the subset of configurations that can be configured by the operator.
//
// https://github.com/open-telemetry/opentelemetry-operator/blob/v0.157.0/cmd/otel-allocator/internal/config/config.go
type Config struct {
	ListenAddr         string                `json:"listen_addr"`
	AllocationStrategy string                `json:"allocation_strategy"`
	CollectorNamespace string                `json:"collector_namespace"`
	CollectorSelector  *metav1.LabelSelector `json:"collector_selector,omitempty"`
	FilterStrategy     string                `json:"filter_strategy"`
	HTTPS              *HTTPSConfig          `json:"https,omitempty"`
	PrometheusCR       ScrapeConfig          `json:"prometheus_cr"`
	Telemetry          *TelemetryConfig      `json:"telemetry,omitempty"`
}

type HTTPSConfig struct {
	Enabled         bool   `json:"enabled"`
	ListenAddr      string `json:"listen_addr"`
	TLSCertFilePath string `json:"tls_cert_file_path"`
	TLSKeyFilePath  string `json:"tls_key_file_path"`
	CAFilePath      string `json:"ca_file_path"`
}

type ScrapeConfig struct {
	Enabled                         bool                  `json:"enabled"`
	ScrapeInterval                  *metav1.Duration      `json:"scrape_interval"`
	PodMonitorSelector              *metav1.LabelSelector `json:"pod_monitor_selector,omitempty"`
	PodMonitorNamespaceSelector     *metav1.LabelSelector `json:"pod_monitor_namespace_selector,omitempty"`
	ServiceMonitorSelector          *metav1.LabelSelector `json:"service_monitor_selector,omitempty"`
	ServiceMonitorNamespaceSelector *metav1.LabelSelector `json:"service_monitor_namespace_selector,omitempty"`
	ScrapeConfigSelector            *metav1.LabelSelector `json:"scrape_config_selector,omitempty"`
	ScrapeConfigNamespaceSelector   *metav1.LabelSelector `json:"scrape_config_namespace_selector,omitempty"`
	ProbeSelector                   *metav1.LabelSelector `json:"probe_selector,omitempty"`
	ProbeNamespaceSelector          *metav1.LabelSelector `json:"probe_namespace_selector,omitempty"`
}

// TelemetryConfig configures the target allocator's own telemetry (OpenTelemetry SDK declarative configuration).
type TelemetryConfig struct {
	Metrics MetricsTelemetryConfig `json:"metrics"`
}

type MetricsTelemetryConfig struct {
	Level   string                `json:"level"`
	Readers []MetricsReaderConfig `json:"readers"`
}

type MetricsReaderConfig struct {
	Periodic PeriodicReaderConfig `json:"periodic"`
}

type PeriodicReaderConfig struct {
	Exporter MetricsExporterConfig `json:"exporter"`
}

type MetricsExporterConfig struct {
	OTLPHTTP OTLPHTTPExporterConfig `json:"otlp_http"`
}

type OTLPHTTPExporterConfig struct {
	Endpoint              string `json:"endpoint"`
	TemporalityPreference string `json:"temporality_preference"`
}

// buildTelemetry exports the target allocator's metrics to the self-monitoring collector of the DynaKube.
// It returns nil if self-monitoring is not enabled.
func buildTelemetry(dk *dynakube.DynaKube) *TelemetryConfig {
	if !dk.IsSelfMonitoringEnabled() {
		return nil
	}

	return &TelemetryConfig{
		Metrics: MetricsTelemetryConfig{
			Level: "normal",
			Readers: []MetricsReaderConfig{{
				Periodic: PeriodicReaderConfig{
					Exporter: MetricsExporterConfig{
						OTLPHTTP: OTLPHTTPExporterConfig{
							Endpoint:              dk.SelfMonitoringOTLPEndpoint(),
							TemporalityPreference: "delta",
						},
					},
				},
			}},
		},
	}
}
