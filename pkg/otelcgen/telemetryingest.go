// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"time"

	"go.opentelemetry.io/collector/pipeline"
)

// TelemetryIngestSpec holds the inputs of the OTel Collector config deployed for telemetryIngest.
// The values may contain `${env:...}` references that the collector resolves when it starts.
type TelemetryIngestSpec struct {
	// Protocols are the enabled receivers, which also decide the pipelines.
	Protocols []Protocol
	// Endpoint is the OTLP endpoint of the exporter.
	Endpoint string
	// APIToken is sent in the Authorization header of the exporter.
	APIToken string
	// CAFile is the CA bundle the exporter trusts. Empty means the exporter does not set a CA.
	CAFile string
	// IncludeSystemCAs additionally makes the exporter trust the system CA pool, if CAFile is set.
	IncludeSystemCAs bool
	// TLSCert and TLSKey enable TLS on the receivers that support it. Both empty means no TLS.
	TLSCert, TLSKey string
	// ResourceAttributes are static resource attributes added to all telemetry.
	ResourceAttributes map[string]string
}

// NewTelemetryIngest builds the OTel Collector config for telemetryIngest.
func NewTelemetryIngest(spec TelemetryIngestSpec) *Config {
	b := NewBuilder()

	for _, protocol := range spec.Protocols {
		b.ReceiverForProtocol(protocol, ReceiverOpts{
			CertFile: spec.TLSCert,
			KeyFile:  spec.TLSKey,
		})
	}

	b.
		telemetryIngestProcessors(spec).
		OTLPHTTPExporter(spec.Endpoint, ExporterOpts{
			CAFile:                   spec.CAFile,
			APIToken:                 spec.APIToken,
			IncludeSystemCACertsPool: spec.IncludeSystemCAs,
		}).
		HealthCheck()

	return b.Build()
}

func (b *Builder) telemetryIngestProcessors(spec TelemetryIngestSpec) *Builder {
	return b.
		MemoryLimiter(MemoryLimiterOpts{LimitPercentage: 70, SpikePercentage: 30}).
		Processor(transformPodIP, buildTransformPodIP()).
		K8sEnrichment(K8sEnrichmentOpts{
			ResourceAttributes: spec.ResourceAttributes,
			// TODO: decide whether we should keep these here
			ResourceStatements: clusterAttributeStatements(),
		}).
		Processor(CumulativeToDeltaID, Empty, pipeline.SignalMetrics).
		Processor(batchTraces, &batchProcessorConfig{
			SendBatchSize:    5000,
			SendBatchMaxSize: 5000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalTraces).
		Processor(batchMetrics, &batchProcessorConfig{
			SendBatchSize:    3000,
			SendBatchMaxSize: 3000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalMetrics).
		Processor(batchLogs, &batchProcessorConfig{
			SendBatchSize:    1800,
			SendBatchMaxSize: 2000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalLogs)
}
