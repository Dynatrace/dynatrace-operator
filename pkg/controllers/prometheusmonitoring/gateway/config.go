// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcfg"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// Component IDs are typed constants so that MustNewID validates the names at program start,
// and the string representations derive from a single source.
// The IDs of the components shared with the other collectors come from otelcfg.
var (
	metricStartTimeID = component.MustNewID("metric_start_time")
)

// prometheusLabelRenames maps the semantic convention name of an attribute to the name Prometheus
// exposes it as, because the characters of a Prometheus label name are restricted.
var prometheusLabelRenames = []struct{ semconv, prometheus string }{
	{"http.request.method", "http_request_method"},
	{"http.response.status_code", "http_response_status_code"},
	{"network.protocol.name", "network_protocol_name"},
	{"network.protocol.version", "network_protocol_version"},
	{"rpc.method", "rpc_method"},
	{"rpc.response.status_code", "rpc_response_status_code"},
	{"rpc.system.name", "rpc_system_name"},
	{"server.address", "server_address"},
	{"server.port", "server_port"},
	{"url.scheme", "url_scheme"},
}

func buildConfig(data gatewayConfigData) *otelcfg.Config {
	return otelcfg.NewBuilder().
		OTLPReceiver("grpc", otelcfg.OTLPReceiverGRPCPort).
		MemoryLimiter(otelcfg.PrometheusMonitoringMemoryLimiter).
		Processor(metricStartTimeID, otelcfg.Empty).
		Processor(otelcfg.CumulativeToDeltaID, map[string]any{
			"initial_value": "drop",
			"max_staleness": "10m",
		}).
		K8sEnrichment(otelcfg.K8sEnrichmentOpts{
			ResourceAttributes: data.ResourceAttributes,
			WorkloadUIDs:       true,
			// k8s.container.name is only extracted by the gateway.
			ExtraFactsMetadata: []string{"k8s.container.name"},
			Signals:            []pipeline.Signal{pipeline.SignalMetrics},
			ResourceStatements: append(scrubOTelScope(), clusterNameFallback),
			Groups: []map[string]any{{
				"context":    "datapoint",
				"statements": prometheusLabelsToSemconv(),
			}},
		}).
		OTLPHTTPExporter(data.Endpoint, otelcfg.ExporterOpts{
			CAFile:          data.CustomCAPath,
			BearerTokenAuth: true,
		}).
		HealthCheck().
		BearerTokenAuth(tokenMountPath + "/" + tokenFileName).
		SelfMonitoring(data.SelfMonitoringEndpoint).
		Build(pipeline.SignalMetrics)
}

// clusterNameFallback sets the cluster name from the environment for telemetry that does not carry it.
var clusterNameFallback = otelcfg.SetAttrFromEnv("k8s.cluster.name", "K8S_CLUSTER_NAME")

// prometheusLabelsToSemconv renames the attributes that Prometheus exposes with underscores to the names of the
// semantic conventions. These only exist on scraped metrics.
func prometheusLabelsToSemconv() []string {
	statements := make([]string, 0, len(prometheusLabelRenames)*2) //nolint:mnd // OTTL rename is 2 statements per entry

	for _, rename := range prometheusLabelRenames {
		statements = append(statements, otelcfg.RenameAttr(rename.semconv, rename.prometheus)...)
	}

	return statements
}

// scrubOTelScope deletes the attributes that describe how the metrics were scraped, which are of no use downstream.
// They must not be deleted for telemetry that is sent to the collector directly, as it can legitimately carry them.
func scrubOTelScope() []string {
	return otelcfg.DeleteKeys("processor", "otel.signal", "otel.scope.name", "otel.scope.version")
}
